// Package web serves the embedded SvelteKit PM UI and JSON state API.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
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
	repoRoot string
	mu       sync.Mutex
	mux      *http.ServeMux
}

// NewServer creates a new ATeam HTTP server rooted at repoRoot.
func NewServer(repoRoot string) (*Server, error) {
	s := &Server{
		repoRoot: repoRoot,
		mux:      http.NewServeMux(),
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
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var card state.BoardCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		writeError(w, http.StatusBadRequest, "invalid card JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := state.Load(s.repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, err := st.UpsertCard(card)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := state.Save(s.repoRoot, st); err != nil {
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
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid move JSON: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := state.Load(s.repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, err := st.MoveCard(req.ID, req.Column, req.Agent)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := state.Save(s.repoRoot, st); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleHandoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
	ho.Branch = st.Git.Branch
	ho.CheckpointSHA = st.Git.HeadSHA
	ho.Dirty = st.Git.Dirty
	ho.UpdatedAt = time.Now().UTC()
	if strings.TrimSpace(ho.Addresses) == "" {
		ho.Addresses = fmt.Sprintf("branch %s, SHA %s", st.Git.Branch, st.Git.HeadSHA)
	}
	st.LastHandoff = &ho
	if err := state.Save(s.repoRoot, st); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st.LastHandoff)
}

type modeRequest struct {
	Mode string `json:"mode"`
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	st, err := state.Load(s.repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	st.Mode = mode
	if err := state.Save(s.repoRoot, st); err != nil {
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
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req syncRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

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
	if len(cards) > 0 {
		st.Cards = cards
		_ = state.Save(s.repoRoot, st)
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
