package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func restoredRequest(s *Server, method, endpoint, body, repo string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, endpoint, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	if repo != "" {
		req.Header.Set("X-ATeam-Repo", repo)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestRepositoryPersistenceFailure(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	primary, other := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "AGENTS.md"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(primary, ".ateam", "repos.json"), 0755); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(primary)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"add_path": other})
	rec := restoredRequest(s, "POST", "/api/repos", string(body), primary)
	if rec.Code != 500 {
		t.Fatalf("persistence failure status = %d, want 500", rec.Code)
	}
	if s.currentRepo() != primary || len(s.repos) != 1 {
		t.Fatalf("failed save changed in-memory selection: %v %s", s.repos, s.currentRepo())
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestRepositorySelectionGuardsMutations(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	primary, other := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "AGENTS.md"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(primary)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"add_path": other})
	if r := restoredRequest(s, "POST", "/api/repos", string(body), primary); r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	for _, req := range []struct{ endpoint, body string }{
		{"/api/cards", `{"id":"LOCAL-1","title":"wrong repo","column":"backlog"}`},
		{"/api/cards/move", `{"id":"LOCAL-1","column":"done"}`},
		{"/api/mode", `{"mode":"off"}`},
		{"/api/handoff", `{"goal":"wrong repo"}`},
		{"/api/forges", `{"provider":"github","owner":"wrong","repo":"repo"}`},
		{"/api/sync", `{"provider":"local"}`},
	} {
		for _, expected := range []string{primary, ""} {
			r := restoredRequest(s, "POST", req.endpoint, req.body, expected)
			if r.Code != 409 {
				t.Errorf("%s stale selection status = %d, want 409", req.endpoint, r.Code)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(other, ".ateam", "state.json")); !os.IsNotExist(err) {
		t.Fatalf("stale request wrote selected repo state: %v", err)
	}
	r := restoredRequest(s, "POST", "/api/mode", `{"mode":"off"}`, other)
	if r.Code != 200 {
		t.Fatalf("current selection refused: %d %s", r.Code, r.Body.String())
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestRestoredDashboardPayload(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	dir := t.TempDir()
	s, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	rec := restoredRequest(s, "GET", "/api/state", "", "")
	if rec.Code != 200 {
		t.Fatalf("dashboard: %s", rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["active_repo"] != dir || len(got["repos"].([]any)) != 1 {
		t.Fatalf("missing repository switcher state: %v", got)
	}
	if _, ok := got["benchmarks"]; ok {
		t.Fatal("dashboard must not execute benchmarks implicitly")
	}
	if got["model_routing"] == nil {
		t.Fatal("restored dashboard lost model routing")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestRepositoryRegistrationHistory(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	primary, other := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "AGENTS.md"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(primary)
	if err != nil {
		t.Fatal(err)
	}
	request := func(action, path string, want int) {
		t.Helper()
		body, _ := json.Marshal(map[string]string{action: path})
		r := restoredRequest(s, "POST", "/api/repos", string(body), s.currentRepo())
		if r.Code != want {
			t.Fatalf("%s status = %d, want %d: %s", action, r.Code, want, r.Body.String())
		}
	}
	request("add_path", other, 200)
	request("add_path", other, 200)
	if len(s.repos) != 2 {
		t.Fatalf("duplicate registration: %v", s.repos)
	}
	s, err = NewServer(primary)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.repos) != 2 || s.currentRepo() != primary {
		t.Fatalf("restart state = %+v, %s", s.repos, s.currentRepo())
	}
	request("active", other, 200)
	request("remove_path", primary, 400)
	request("remove_path", other, 200)
	if s.currentRepo() != primary {
		t.Fatalf("removed active selection did not fall back: %s", s.currentRepo())
	}
	s, err = NewServer(primary)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.repos) != 1 {
		t.Fatalf("removed repo returned after restart: %v", s.repos)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestRestoredSettingsAuthorization(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "test-secret")
	s, err := NewServer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"/api/repos", "/api/forges"} {
		for _, method := range []string{"GET", "POST"} {
			if endpoint == "/api/forges" && method == "GET" {
				continue
			}
			r := restoredRequest(s, method, endpoint, `{}`, "")
			if r.Code != 401 {
				t.Errorf("%s %s unauthenticated = %d", method, endpoint, r.Code)
			}
		}
		req := httptest.NewRequest("DELETE", endpoint, nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("X-ATeam-Token", "test-secret")
		r := httptest.NewRecorder()
		s.Handler().ServeHTTP(r, req)
		if r.Code != 405 {
			t.Errorf("DELETE %s = %d, want 405", endpoint, r.Code)
		}
	}
}
