package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

func TestServerEndpoints(t *testing.T) {
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	h := srv.Handler()

	// 1. GET /api/state
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/state status = %d, want 200", rec.Code)
	}

	// 2. POST /api/cards
	cardBody, _ := json.Marshal(state.BoardCard{
		ID:     "ATEAM-10",
		Title:  "Build SvelteKit Kanban view",
		Column: "backlog",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/cards", bytes.NewReader(cardBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/cards status = %d: %s", rec.Code, rec.Body.String())
	}

	// 3. POST /api/cards/move
	moveBody, _ := json.Marshal(map[string]string{
		"id":     "ATEAM-10",
		"column": "in-progress",
		"agent":  "claude",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/cards/move", bytes.NewReader(moveBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/cards/move status = %d: %s", rec.Code, rec.Body.String())
	}

	// 4. POST /api/handoff
	hoBody, _ := json.Marshal(state.Handoff{
		Goal:        "Build UI",
		Addresses:   "ATEAM-10",
		Constraints: "zero runtime deps",
		DoneWhen:    "task verify green",
		ChecksToRun: "task verify",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/handoff", bytes.NewReader(hoBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/handoff status = %d: %s", rec.Code, rec.Body.String())
	}

	// 5. POST /api/mode
	modeBody, _ := json.Marshal(map[string]string{"mode": "ultra"})
	req = httptest.NewRequest(http.MethodPost, "/api/mode", bytes.NewReader(modeBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/mode status = %d: %s", rec.Code, rec.Body.String())
	}

	// 6. GET / (SPA static index)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", rec.Code)
	}
}
