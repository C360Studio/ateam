package web

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestServerEndpoints(t *testing.T) {
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	h := srv.Handler()

	// 1. GET /api/state (loopback)
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	req.RemoteAddr = "127.0.0.1:12345"
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
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
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
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
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
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/handoff status = %d: %s", rec.Code, rec.Body.String())
	}

	// 5. POST /api/mode
	modeBody, _ := json.Marshal(map[string]string{"mode": "ultra"})
	req = httptest.NewRequest(http.MethodPost, "/api/mode", bytes.NewReader(modeBody))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
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

// spec: openspec/specs/harness/spec.md#Requirement: Constant-time forge webhook verification and rate limiting
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

// spec: openspec/specs/harness/spec.md#Requirement: Constant-time forge webhook verification and rate limiting
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestSecurityAndFailClosedProbes(t *testing.T) {
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	h := srv.Handler()

	// Probe 1a: Unsigned webhook when ATEAM_WEBHOOK_SECRET is unset must be rejected (401).
	unsignedPayload := []byte(`{"action":"opened","issue":{"number":7,"title":"Unsigned issue","state":"open"}}`)
	reqUnsigned := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", bytes.NewReader(unsignedPayload))
	recUnsigned := httptest.NewRecorder()
	h.ServeHTTP(recUnsigned, reqUnsigned)
	if recUnsigned.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned webhook without secret status = %d, want 401", recUnsigned.Code)
	}

	// Probe 1b: Cross-origin Origin header and text/plain Content-Type must be rejected on /api/mode.
	modeBody := []byte(`{"mode":"off"}`)
	reqPlain := httptest.NewRequest(http.MethodPost, "/api/mode", bytes.NewReader(modeBody))
	reqPlain.RemoteAddr = "127.0.0.1:12345"
	reqPlain.Header.Set("Content-Type", "text/plain")
	recPlain := httptest.NewRecorder()
	h.ServeHTTP(recPlain, reqPlain)
	if recPlain.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("POST /api/mode with text/plain status = %d, want 415", recPlain.Code)
	}

	reqEvilOrigin := httptest.NewRequest(http.MethodPost, "/api/mode", bytes.NewReader(modeBody))
	reqEvilOrigin.RemoteAddr = "127.0.0.1:12345"
	reqEvilOrigin.Header.Set("Content-Type", "application/json")
	reqEvilOrigin.Header.Set("Origin", "https://evil.example")
	recEvilOrigin := httptest.NewRecorder()
	h.ServeHTTP(recEvilOrigin, reqEvilOrigin)
	if recEvilOrigin.Code != http.StatusForbidden {
		t.Fatalf("POST /api/mode with evil Origin status = %d, want 403", recEvilOrigin.Code)
	}

	st, err := state.Load(dir)
	if err != nil {
		t.Fatalf("state.Load: %v", err)
	}
	if st.Mode == state.ModeOff {
		t.Fatal("cross-origin or text/plain request mutated state.Mode to off")
	}

	// Probe 6: Invalid UTF-8 tasks.md with Hold must fail closed instead of returning merge_allowed: true.
	srv.webhookSecret = "secret-2"
	changeDir := filepath.Join(dir, "openspec", "changes", "broken-utf8")
	_ = os.MkdirAll(changeDir, 0o755)
	_ = os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Work\n- [ ] 1.1 Task\n  - Hold: blocked \xff\xfe\n"), 0o644)

	mac := hmac.New(sha256.New, []byte(srv.webhookSecret))
	mac.Write(unsignedPayload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	reqSigned := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", bytes.NewReader(unsignedPayload))
	reqSigned.Header.Set("X-Hub-Signature-256", sig)
	recSigned := httptest.NewRecorder()
	h.ServeHTTP(recSigned, reqSigned)
	if recSigned.Code == http.StatusOK {
		t.Fatalf("expected invalid UTF-8 Hold scan to fail closed, got 200: %s", recSigned.Body.String())
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestReadAuthAndLoopbackEnforcement(t *testing.T) {
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	h := srv.Handler()

	// Astra #5: 192.0.2.1:1234 (non-loopback) must be rejected when ATEAM_API_TOKEN is unset.
	for _, path := range []string{"/api/state", "/api/openspec", "/api/shortcuts", "/api/benchmarks"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s from 192.0.2.1 without token status = %d, want 401", path, rec.Code)
		}
	}

	reqMut := httptest.NewRequest(http.MethodPost, "/api/mode", bytes.NewReader([]byte(`{"mode":"lite"}`)))
	reqMut.RemoteAddr = "192.0.2.1:1234"
	reqMut.Header.Set("Content-Type", "application/json")
	recMut := httptest.NewRecorder()
	h.ServeHTTP(recMut, reqMut)
	if recMut.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/mode from 192.0.2.1 without token status = %d, want 401", recMut.Code)
	}

	// Astra #1 & #2: When ATEAM_API_TOKEN is set, even loopback GET/POST require X-ATeam-Token.
	srv.apiToken = "tok-secret-99"
	reqNoTok := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	reqNoTok.RemoteAddr = "127.0.0.1:12345"
	recNoTok := httptest.NewRecorder()
	h.ServeHTTP(recNoTok, reqNoTok)
	if recNoTok.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/state without token when apiToken set status = %d, want 401", recNoTok.Code)
	}

	reqWithTok := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	reqWithTok.RemoteAddr = "203.0.113.5:9000"
	reqWithTok.Header.Set("X-ATeam-Token", "tok-secret-99")
	recWithTok := httptest.NewRecorder()
	h.ServeHTTP(recWithTok, reqWithTok)
	if recWithTok.Code != http.StatusOK {
		t.Fatalf("GET /api/state with valid X-ATeam-Token status = %d, want 200", recWithTok.Code)
	}
}
