// Package web serves the embedded SvelteKit PM UI and JSON state API.
package web

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/c360studio/ateam/internal/bench"
	"github.com/c360studio/ateam/internal/state"
	"github.com/c360studio/ateam/internal/taskrunner"
)

//go:embed all:dist
var embeddedDist embed.FS

// RepoEntry describes a repository registered with the ATeam web server.
type RepoEntry struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Active bool   `json:"active"`
}

// DashboardPayload aggregates the full state for the SvelteKit PM UI.
type DashboardPayload struct {
	ModelRouting ModelRouting        `json:"model_routing"`
	State        *state.State        `json:"state"`
	PickupText   string              `json:"pickup_text"`
	Benchmarks   *bench.SuiteReport  `json:"benchmarks,omitempty"`
	Tasks        []taskrunner.Target `json:"tasks"`
	Repos        []RepoEntry         `json:"repos"`
	ActiveRepo   string              `json:"active_repo"`
}

// Server serves the ATeam HTTP API and embedded static UI.
type Server struct {
	repoRoot              string
	activeRepo            string
	repos                 []string
	webhookSecret         string
	allowUnsignedWebhooks bool
	apiToken              string
	limiter               *webhookLimiter
	mu                    sync.Mutex
	mux                   *http.ServeMux
}

// NewServer creates a new ATeam HTTP server rooted at repoRoot.
func NewServer(repoRoot string) (*Server, error) {
	cleanRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}
	s := &Server{
		repoRoot:              cleanRoot,
		activeRepo:            cleanRoot,
		repos:                 loadRegisteredRepos(cleanRoot),
		webhookSecret:         strings.TrimSpace(os.Getenv("ATEAM_WEBHOOK_SECRET")),
		allowUnsignedWebhooks: strings.TrimSpace(os.Getenv("ATEAM_ALLOW_UNSIGNED_WEBHOOKS")) == "1",
		apiToken:              strings.TrimSpace(os.Getenv("ATEAM_API_TOKEN")),
		limiter:               newWebhookLimiter(60, time.Minute),
		mux:                   http.NewServeMux(),
	}
	if err := s.registerRoutes(); err != nil {
		return nil, err
	}
	return s, nil
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() error {
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/model-routing", s.handleModelRouting)
	s.mux.HandleFunc("/api/repos", s.handleRepos)
	s.mux.HandleFunc("/api/forges", s.handleForges)
	s.mux.HandleFunc("/api/cards", s.handleCards)
	s.mux.HandleFunc("/api/cards/move", s.handleCardMove)
	s.mux.HandleFunc("/api/handoff", s.handleHandoff)
	s.mux.HandleFunc("/api/mode", s.handleMode)
	s.mux.HandleFunc("/api/openspec", s.handleOpenSpec)
	s.mux.HandleFunc("/api/shortcuts", s.handleShortcuts)
	s.mux.HandleFunc("/api/benchmarks", s.handleBenchmarks)
	s.mux.HandleFunc("/api/sync", s.handleSyncForge)
	s.mux.HandleFunc("/api/webhooks/gitea", s.handleWebhook("gitea"))
	s.mux.HandleFunc("/api/webhooks/github", s.handleWebhook("github"))

	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(sub))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(sub, path); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
	return nil
}

func (s *Server) authorizeAPIRequest(w http.ResponseWriter, r *http.Request, remoteMsg string) bool {
	token := strings.TrimSpace(s.apiToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("ATEAM_API_TOKEN"))
	}
	if token != "" {
		provided := strings.TrimSpace(r.Header.Get("X-ATeam-Token"))
		if provided == "" {
			if bearer, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer "); ok {
				provided = strings.TrimSpace(bearer)
			}
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid or missing API token")
			return false
		}
		return true
	}
	if !isLoopbackRequest(r.RemoteAddr) {
		writeError(w, http.StatusUnauthorized, remoteMsg)
		return false
	}
	return true
}

func (s *Server) validateReadRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return s.authorizeAPIRequest(w, r, "remote read requires ATEAM_API_TOKEN")
}

func (s *Server) validateMutationRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")), "cross-site") {
		writeError(w, http.StatusForbidden, "cross-origin mutation rejected")
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || !strings.EqualFold(u.Host, r.Host) {
			writeError(w, http.StatusForbidden, "cross-origin mutation rejected")
			return false
		}
	}
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.HasPrefix(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	return s.authorizeAPIRequest(w, r, "remote mutation requires ATEAM_API_TOKEN")
}

func isLoopbackRequest(remoteAddr string) bool {
	host := extractClientIP(remoteAddr)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func loadRegisteredRepos(primary string) []string {
	repos := []string{primary}
	seen := map[string]bool{primary: true}
	p := filepath.Join(primary, ".ateam", "repos.json")
	if data, err := os.ReadFile(p); err == nil {
		var saved []string
		if json.Unmarshal(data, &saved) == nil {
			for _, item := range saved {
				clean := filepath.Clean(strings.TrimSpace(item))
				if clean != "" && !seen[clean] && isValidRepoDir(clean) {
					seen[clean] = true
					repos = append(repos, clean)
				}
			}
		}
	}
	return repos
}

func saveRegisteredRepos(primary string, repos []string) error {
	dir := filepath.Join(primary, ".ateam")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create repository registry: %w", err)
	}
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "repos-*.json")
	if err != nil {
		return fmt.Errorf("create repository registry: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return fmt.Errorf("write repository registry: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close repository registry: %w", err)
	}
	if err := os.Rename(file.Name(), filepath.Join(dir, "repos.json")); err != nil {
		return fmt.Errorf("save repository registry: %w", err)
	}
	return nil
}

func isValidRepoDir(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		return true
	}
	return false
}

func (s *Server) currentRepo() string {
	if s.activeRepo != "" {
		return s.activeRepo
	}
	return s.repoRoot
}

func (s *Server) repoEntries() []RepoEntry {
	active := s.currentRepo()
	out := make([]RepoEntry, 0, len(s.repos))
	for _, p := range s.repos {
		name := filepath.Base(p)
		out = append(out, RepoEntry{
			ID:     p,
			Name:   name,
			Path:   p,
			Active: p == active,
		})
	}
	return out
}

func (s *Server) buildDashboard(ctx context.Context) (*DashboardPayload, error) {
	root := s.currentRepo()
	st, err := state.Refresh(ctx, root, false)
	if err != nil {
		return nil, err
	}
	tasks, _ := taskrunner.ListTargets(root)
	return &DashboardPayload{
		ModelRouting: readModelRouting(root),
		State:        st,
		PickupText:   state.BuildPickupSummary(st),
		Tasks:        tasks,
		Repos:        s.repoEntries(),
		ActiveRepo:   root,
	}, nil
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.validateReadRequest(w, r) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	payload, err := s.buildDashboard(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

type repoRequest struct {
	Active     string `json:"active,omitempty"`
	AddPath    string `json:"add_path,omitempty"`
	RemovePath string `json:"remove_path,omitempty"`
}

func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if !s.validateReadRequest(w, r) {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"repos":       s.repoEntries(),
			"active_repo": s.currentRepo(),
		})
		return
	}
	if !s.validateMutationRequest(w, r) {
		return
	}
	var req repoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if add := strings.TrimSpace(req.AddPath); add != "" {
		clean, err := filepath.Abs(add)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid repository path")
			return
		}
		if !isValidRepoDir(clean) {
			writeError(w, http.StatusBadRequest, "directory must exist and contain .git or AGENTS.md")
			return
		}
		exists := false
		for _, p := range s.repos {
			if p == clean {
				exists = true
				break
			}
		}
		if !exists {
			next := append(append([]string(nil), s.repos...), clean)
			if err := saveRegisteredRepos(s.repoRoot, next); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			s.repos = next
		}
		s.activeRepo = clean
	} else if rem := strings.TrimSpace(req.RemovePath); rem != "" {
		clean := filepath.Clean(rem)
		if clean == s.repoRoot {
			writeError(w, http.StatusBadRequest, "cannot remove primary startup repository")
			return
		}
		var kept []string
		for _, p := range s.repos {
			if p != clean {
				kept = append(kept, p)
			}
		}
		if err := saveRegisteredRepos(s.repoRoot, kept); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.repos = kept
		if s.activeRepo == clean {
			s.activeRepo = s.repoRoot
		}
	} else if act := strings.TrimSpace(req.Active); act != "" {
		matched := ""
		clean := filepath.Clean(act)
		for _, p := range s.repos {
			if p == clean {
				matched = p
				break
			}
		}
		if matched == "" {
			writeError(w, http.StatusBadRequest, "unknown repository: "+act)
			return
		}
		s.activeRepo = matched
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"repos":       s.repoEntries(),
		"active_repo": s.currentRepo(),
	})
}

type forgeRequest struct {
	Action   string              `json:"action,omitempty"`
	Provider state.ForgeProvider `json:"provider"`
	BaseURL  string              `json:"base_url,omitempty"`
	Owner    string              `json:"owner,omitempty"`
	Repo     string              `json:"repo,omitempty"`
}

func (s *Server) handleForges(w http.ResponseWriter, r *http.Request) {
	if !s.validateMutationRequest(w, r) {
		return
	}
	var req forgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid forge JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validateRepoSelection(w, r) {
		return
	}

	root := s.currentRepo()
	var badReqErr error
	st, err := state.Update(root, func(cur *state.State) error {
		if strings.EqualFold(strings.TrimSpace(req.Action), "remove") {
			cur.RemoveForge(req.Provider)
			return nil
		}
		if err := cur.UpsertForge(state.ForgeConfig{
			Provider: req.Provider,
			BaseURL:  req.BaseURL,
			Owner:    req.Owner,
			Repo:     req.Repo,
		}); err != nil {
			badReqErr = err
			return err
		}
		return nil
	})
	if badReqErr != nil {
		writeError(w, http.StatusBadRequest, badReqErr.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"forge":  st.Forge,
		"forges": st.Forges,
	})
}

func (s *Server) handleCards(w http.ResponseWriter, r *http.Request) {
	if !s.validateMutationRequest(w, r) {
		return
	}
	var card state.BoardCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		writeError(w, http.StatusBadRequest, "invalid card JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validateRepoSelection(w, r) {
		return
	}

	var updated state.BoardCard
	var badReqErr error
	_, err := state.Update(s.currentRepo(), func(st *state.State) error {
		c, err := st.UpsertCard(card)
		if err != nil {
			badReqErr = err
			return err
		}
		updated = c
		return nil
	})
	if badReqErr != nil {
		writeError(w, http.StatusBadRequest, badReqErr.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type moveRequest struct {
	ID     string `json:"id"`
	Column string `json:"column"`
	Agent  string `json:"agent,omitempty"`
}

func (s *Server) handleCardMove(w http.ResponseWriter, r *http.Request) {
	if !s.validateMutationRequest(w, r) {
		return
	}
	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid move JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validateRepoSelection(w, r) {
		return
	}

	var updated state.BoardCard
	var badReqErr error
	_, err := state.Update(s.currentRepo(), func(st *state.State) error {
		c, err := st.MoveCard(req.ID, req.Column, req.Agent)
		if err != nil {
			badReqErr = err
			return err
		}
		updated = c
		return nil
	})
	if badReqErr != nil {
		writeError(w, http.StatusBadRequest, badReqErr.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleHandoff(w http.ResponseWriter, r *http.Request) {
	if !s.validateMutationRequest(w, r) {
		return
	}
	var ho state.Handoff
	if err := json.NewDecoder(r.Body).Decode(&ho); err != nil {
		writeError(w, http.StatusBadRequest, "invalid handoff JSON: "+err.Error())
		return
	}
	if strings.TrimSpace(ho.Goal) == "" {
		writeError(w, http.StatusBadRequest, "goal is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validateRepoSelection(w, r) {
		return
	}

	root := s.currentRepo()
	st, err := state.Refresh(r.Context(), root, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resolvedSession := state.ResolveActiveSessionID(root, ho.SessionID)
	resolvedAgent := state.ResolveActiveAgent(root, ho.Agent, st.Git.Branch, st.Writer)
	var saved state.Handoff
	_, err = state.Update(root, func(cur *state.State) error {
		cur.CurrentAgent = resolvedAgent
		ho.Agent = resolvedAgent
		ho.SessionID = resolvedSession
		ho.Branch = st.Git.Branch
		ho.CheckpointSHA = st.Git.HeadSHA
		ho.Dirty = st.Git.Dirty
		ho.UpdatedAt = time.Now().UTC()
		if strings.TrimSpace(ho.Addresses) == "" {
			ho.Addresses = fmt.Sprintf("branch %s, SHA %s", st.Git.Branch, st.Git.HeadSHA)
		}
		cur.SetHandoff(ho)
		saved = *cur.LastHandoff
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

type modeRequest struct {
	Mode string `json:"mode"`
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	if !s.validateMutationRequest(w, r) {
		return
	}
	var req modeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid mode JSON: "+err.Error())
		return
	}
	mode, ok := state.NormalizeMode(req.Mode)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid mode: "+req.Mode)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validateRepoSelection(w, r) {
		return
	}

	st, err := state.Update(s.currentRepo(), func(st *state.State) error {
		st.Mode = mode
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"mode": string(st.Mode)})
}

func (s *Server) handleOpenSpec(w http.ResponseWriter, r *http.Request) {
	if !s.validateReadRequest(w, r) {
		return
	}
	s.mu.Lock()
	root := s.currentRepo()
	s.mu.Unlock()
	changes, err := state.ScanOpenSpecQueue(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	specs, _ := state.ScanOpenSpecSpecs(root)
	formatted, hasCaveats := state.FormatOpenSpecQueue(changes)
	writeJSON(w, http.StatusOK, map[string]any{
		"specs":       specs,
		"changes":     changes,
		"formatted":   formatted,
		"has_caveats": hasCaveats,
	})
}

func (s *Server) handleShortcuts(w http.ResponseWriter, r *http.Request) {
	if !s.validateReadRequest(w, r) {
		return
	}
	s.mu.Lock()
	root := s.currentRepo()
	s.mu.Unlock()
	items, err := state.ScanShortcuts(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleBenchmarks(w http.ResponseWriter, r *http.Request) {
	if !s.validateReadRequest(w, r) {
		return
	}
	report, err := bench.RunSelfTest(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

type syncRequest struct {
	Provider state.ForgeProvider `json:"provider"`
}

func (s *Server) handleSyncForge(w http.ResponseWriter, r *http.Request) {
	if !s.validateMutationRequest(w, r) {
		return
	}
	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid sync JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.validateRepoSelection(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	root := s.currentRepo()
	st, err := state.Load(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg := st.ResolveForge(req.Provider)
	if cfg.Provider == "" || cfg.Provider == state.ForgeLocal {
		writeError(w, http.StatusBadRequest, "forge is not configured for this repository")
		return
	}
	cards, err := state.SyncForgeBoard(ctx, root, cfg)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if _, err := state.Update(root, func(cur *state.State) error {
		cur.MergeForgeCards(cards, cfg.Provider)
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"synced": len(cards), "forge": cfg})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// validateRepoSelection must run under mu before any selected-repository write.
// Legacy single-repository clients may omit the header; multi-repository clients
// must bind mutations to the repository they actually displayed.
func (s *Server) validateRepoSelection(w http.ResponseWriter, r *http.Request) bool {
	expected := r.Header.Get("X-ATeam-Repo")
	if expected == s.currentRepo() || (expected == "" && len(s.repos) == 1) {
		return true
	}
	writeError(w, http.StatusConflict, "active repository changed; refresh before making changes")
	return false
}
