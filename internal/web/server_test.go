package web

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestWebhookSignatureAndRateLimit(t *testing.T) {
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	srv.webhookSecret = "top-secret-key"
	srv.limiter = newWebhookLimiter(2, time.Minute)
	h := srv.Handler()

	payload := []byte(`{"action":"opened","pull_request":{"number":42,"title":"Add Gitea webhook handler","state":"open","draft":false,"head":{"ref":"gemini/webhooks"}}}`)
	mac := hmac.New(sha256.New, []byte(srv.webhookSecret))
	mac.Write(payload)
	sigHex := hex.EncodeToString(mac.Sum(nil))

	// 1. Valid Gitea signature -> 200 OK
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/gitea", bytes.NewReader(payload))
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Gitea-Signature", sigHex)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/webhooks/gitea valid sig status = %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Invalid GitHub signature -> 401 Unauthorized
	reqBad := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", bytes.NewReader(payload))
	reqBad.RemoteAddr = "10.0.0.1:12345"
	reqBad.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	recBad := httptest.NewRecorder()
	h.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/webhooks/github bad sig status = %d, want 401", recBad.Code)
	}

	// 3. Third request from same IP within window -> 429 Too Many Requests
	reqRate := httptest.NewRequest(http.MethodPost, "/api/webhooks/gitea", bytes.NewReader(payload))
	reqRate.RemoteAddr = "10.0.0.1:12345"
	reqRate.Header.Set("X-Gitea-Signature", sigHex)
	recRate := httptest.NewRecorder()
	h.ServeHTTP(recRate, reqRate)
	if recRate.Code != http.StatusTooManyRequests {
		t.Fatalf("POST /api/webhooks/gitea rate-limited status = %d, want 429", recRate.Code)
	}
}
