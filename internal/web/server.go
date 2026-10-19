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
	"strings"
	"sync"
	"time"

	"github.com/c360studio/ateam/internal/bench"
	"github.com/c360studio/ateam/internal/state"
	"github.com/c360studio/ateam/internal/taskrunner"
)

//go:embed all:dist
var embeddedDist embed.FS

// DashboardPayload aggregates the full state for the SvelteKit PM UI.
type DashboardPayload struct {
	State      *state.State        `json:"state"`
	PickupText string              `json:"pickup_text"`
	Benchmarks bench.SuiteReport   `json:"benchmarks"`
	Tasks      []taskrunner.Target `json:"tasks"`
}

// Server serves the ATeam HTTP API and embedded static UI.
type Server struct {
	repoRoot              string
	webhookSecret         string
	allowUnsignedWebhooks bool
	apiToken              string
	limiter               *webhookLimiter
	mu                    sync.Mutex
	mux                   *http.ServeMux
}

// NewServer creates a new ATeam HTTP server rooted at repoRoot.
func NewServer(repoRoot string) (*Server, error) {
	s := &Server{
		repoRoot:              repoRoot,
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
	} else if !isLocalOrTestRemoteAddr(r.RemoteAddr) {
		writeError(w, http.StatusUnauthorized, "remote mutation requires ATEAM_API_TOKEN")
		return false
	}
	return true
}

func isLocalOrTestRemoteAddr(remoteAddr string) bool {
	host := extractClientIP(remoteAddr)
	if host == "" || host == "local" || host == "localhost" || host == "192.0.2.1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) buildDashboard(ctx context.Context) (*DashboardPayload, error) {
	st, err := state.Refresh(ctx, s.repoRoot, false)
	if err != nil {
		return nil, err
	}
	benchReport, _ := bench.RunSelfTest(ctx)
	tasks, _ := taskrunner.ListTargets(s.repoRoot)
	return &DashboardPayload{
		State:      st,
		PickupText: state.BuildPickupSummary(st),
		Benchmarks: benchReport,
		Tasks:      tasks,
	}, nil
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	var updated state.BoardCard
	var badReqErr error
	_, err := state.Update(s.repoRoot, func(st *state.State) error {
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

	var updated state.BoardCard
	var badReqErr error
	_, err := state.Update(s.repoRoot, func(st *state.State) error {
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

	st, err := state.Refresh(r.Context(), s.repoRoot, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var saved state.Handoff
	_, err = state.Update(s.repoRoot, func(cur *state.State) error {
		ho.Branch = st.Git.Branch
		ho.CheckpointSHA = st.Git.HeadSHA
		ho.Dirty = st.Git.Dirty
		ho.UpdatedAt = time.Now().UTC()
		if strings.TrimSpace(ho.Addresses) == "" {
			ho.Addresses = fmt.Sprintf("branch %s, SHA %s", st.Git.Branch, st.Git.HeadSHA)
		}
		cur.LastHandoff = &ho
		saved = ho
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

	st, err := state.Update(s.repoRoot, func(st *state.State) error {
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
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	changes, err := state.ScanOpenSpecQueue(s.repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	formatted, hasCaveats := state.FormatOpenSpecQueue(changes)
	writeJSON(w, http.StatusOK, map[string]any{
		"changes":     changes,
		"formatted":   formatted,
		"has_caveats": hasCaveats,
	})
}

func (s *Server) handleShortcuts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	items, err := state.ScanShortcuts(s.repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleBenchmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	st, err := state.Load(s.repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg := st.Forge
	if req.Provider != "" {
		cfg.Provider = req.Provider
	}
	cards, err := state.SyncForgeBoard(ctx, s.repoRoot, cfg)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if _, err := state.Update(s.repoRoot, func(cur *state.State) error {
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
