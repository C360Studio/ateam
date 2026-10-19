package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const routingPolicyFixture = `{"version":1,"models":[{"id":"dev-model","provider":"codex"},{"id":"review-model","provider":"codex","stronger_than":["dev-model"]}],"routes":{"developer":{"model":"dev-model","effort":"custom-effort"},"reviewer":{"model":"review-model","effort":"high"}},"owners":["owner"],"review_authors":["reviewer"],"required_checks":["task verify"],"cross_provider":false}`

func writeRoutingPolicy(t *testing.T, dir, data string) string {
	t.Helper()
	path := filepath.Join(dir, ".agents", "review-policy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func requestRouting(t *testing.T, srv *Server, endpoint string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, endpoint, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("routing response must prohibit caching")
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("expected routing JSON: %v", err)
	}
	if endpoint == "/api/state" {
		var ok bool
		result, ok = result["model_routing"].(map[string]any)
		if !ok {
			t.Fatal("dashboard missing model_routing")
		}
	}
	return result
}

// spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
func TestModelRoutingStates(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	equal := strings.Replace(routingPolicyFixture, `"model":"review-model"`, `"model":"dev-model"`, 1)
	cases := []struct {
		name, data, status, reason string
		routes                     bool
	}{
		{"absent", "", "unconfigured", "No model routing policy", false},
		{"eligible", routingPolicyFixture, "eligible", "", true},
		{"ceiling despite higher effort", equal, "blocked", "stronger", true},
		{"cross provider required", strings.Replace(routingPolicyFixture, `"cross_provider":false`, `"cross_provider":true`, 1), "blocked", "provider", true},
		{"different provider eligible", strings.Replace(strings.Replace(routingPolicyFixture, `"id":"review-model","provider":"codex"`, `"id":"review-model","provider":"claude"`, 1), `"cross_provider":false`, `"cross_provider":true`, 1), "eligible", "", true},
		{"malformed", "{", "invalid", "Invalid model routing policy", false},
		{"unknown model", strings.Replace(routingPolicyFixture, `"model":"review-model"`, `"model":"unknown"`, 1), "invalid", "unknown model", false},
		{"unresolved effort", strings.Replace(routingPolicyFixture, "custom-effort", "inherit", 1), "invalid", "effort", false},
		{"exact byte limit", routingPolicyFixture + strings.Repeat(" ", (1<<20)-len(routingPolicyFixture)), "eligible", "", true},
		{"over byte limit", routingPolicyFixture + strings.Repeat(" ", (1<<20)+1-len(routingPolicyFixture)), "invalid", "1 MiB", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.data != "" {
				writeRoutingPolicy(t, dir, tc.data)
			}
			srv, err := NewServer(dir)
			if err != nil {
				t.Fatal(err)
			}
			got := requestRouting(t, srv, "/api/model-routing")
			if got["status"] != tc.status {
				t.Fatalf("routing status = %v, want %s", got["status"], tc.status)
			}
			reason, _ := got["reason"].(string)
			if !strings.Contains(reason, tc.reason) {
				t.Errorf("reason = %q, want %q", reason, tc.reason)
			}
			if tc.routes {
				dev, ok := got["developer"].(map[string]any)
				if !ok || dev["model"] != "dev-model" || dev["provider"] != "codex" || dev["effort"] != "custom-effort" {
					t.Fatalf("developer metadata = %#v", dev)
				}
				reviewer, ok := got["reviewer"].(map[string]any)
				if !ok || reviewer["effort"] != "high" {
					t.Fatalf("reviewer metadata = %#v", reviewer)
				}
				wantModel, wantProvider := "review-model", "codex"
				if tc.name == "ceiling despite higher effort" {
					wantModel = "dev-model"
				}
				if tc.name == "different provider eligible" {
					wantProvider = "claude"
				}
				if reviewer["model"] != wantModel || reviewer["provider"] != wantProvider {
					t.Errorf("reviewer = %#v", reviewer)
				}
				digest, _ := got["policy_digest"].(string)
				if len(digest) != 64 {
					t.Errorf("missing policy digest: %v", got)
				}
			} else if got["developer"] != nil || got["reviewer"] != nil || got["policy_digest"] != nil {
				t.Errorf("unusable policy exposed route metadata: %v", got)
			}
			if got["owners"] != nil || got["review_authors"] != nil {
				t.Error("summary leaked full policy")
			}
			if tc.data != "" {
				after, err := os.ReadFile(filepath.Join(dir, ".agents", "review-policy.json"))
				if err != nil {
					t.Fatal(err)
				}
				if string(after) != tc.data {
					t.Fatal("read mutated policy")
				}
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
func TestModelRoutingConfinedRegularFile(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	for _, kind := range []string{"directory", "symlink", "dangling symlink", "escaped parent", "missing root"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := writeRoutingPolicy(t, dir, routingPolicyFixture)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "directory":
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink", "dangling symlink":
				outside := filepath.Join(t.TempDir(), "secret.json")
				if kind == "symlink" {
					if err := os.WriteFile(outside, []byte(routingPolicyFixture), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			case "escaped parent":
				if err := os.Remove(filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
				outside := t.TempDir()
				writeRoutingPolicy(t, outside, routingPolicyFixture)
				if err := os.Symlink(filepath.Join(outside, ".agents"), filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
			case "missing root":
				dir = filepath.Join(dir, "missing")
			}
			srv, err := NewServer(dir)
			if err != nil {
				t.Fatal(err)
			}
			got := requestRouting(t, srv, "/api/model-routing")
			if got["status"] != "unavailable" || got["developer"] != nil {
				t.Fatalf("unsafe path must be unavailable: %v", got)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestModelRoutingAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name, configured, provided, remote, method string
		status                                     int
	}{
		{"local", "", "", "127.0.0.1:1", "GET", 200},
		{"remote denied", "", "", "192.0.2.1:1", "GET", 401},
		{"local token required", "secret", "", "127.0.0.1:1", "GET", 401},
		{"wrong token", "secret", "wrong", "192.0.2.1:1", "GET", 401},
		{"remote authenticated", "secret", "secret", "192.0.2.1:1", "GET", 200},
		{"write refused", "secret", "secret", "127.0.0.1:1", "POST", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATEAM_API_TOKEN", tc.configured)
			srv, err := NewServer(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(tc.method, "/api/model-routing", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("X-ATeam-Token", tc.provided)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
func TestModelRoutingRefreshHistory(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	previousDigest := ""
	for _, step := range []struct{ data, status string }{
		{routingPolicyFixture, "eligible"},
		{strings.Replace(routingPolicyFixture, `"model":"review-model"`, `"model":"dev-model"`, 1), "blocked"},
		{"{", "invalid"}, {"", "unconfigured"}, {routingPolicyFixture, "eligible"},
	} {
		path := writeRoutingPolicy(t, dir, step.data)
		if step.data == "" {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}
		for _, endpoint := range []string{"/api/model-routing", "/api/state"} {
			got := requestRouting(t, srv, endpoint)
			if got["status"] != step.status {
				t.Fatalf("%s: status = %v, want %s", endpoint, got["status"], step.status)
			}
			if endpoint == "/api/model-routing" {
				digest, _ := got["policy_digest"].(string)
				if digest != "" && digest == previousDigest {
					t.Error("changed policy retained old digest")
				}
				previousDigest = digest
			}
		}
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
func TestModelRoutingSelectedRepository(t *testing.T) {
	t.Setenv("ATEAM_API_TOKEN", "")
	primary, secondary, missing := t.TempDir(), t.TempDir(), t.TempDir()
	writeRoutingPolicy(t, primary, routingPolicyFixture)
	writeRoutingPolicy(t, secondary, strings.ReplaceAll(strings.Replace(routingPolicyFixture, `"model":"review-model"`, `"model":"dev-model"`, 1), "dev-model", "secondary-dev"))
	for _, root := range []string{secondary, missing} {
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := NewServer(primary)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct{ action, path, status, model string }{
		{"active", primary, "eligible", "dev-model"},
		{"add_path", secondary, "blocked", "secondary-dev"},
		{"add_path", missing, "unconfigured", ""},
		{"active", secondary, "blocked", "secondary-dev"},
		{"remove_path", secondary, "eligible", "dev-model"},
	} {
		body, _ := json.Marshal(map[string]string{step.action: step.path})
		req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(string(body)))
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("select repo: %d %s", rec.Code, rec.Body.String())
		}
		for _, endpoint := range []string{"/api/state", "/api/model-routing"} {
			got := requestRouting(t, srv, endpoint)
			if got["status"] != step.status {
				t.Errorf("%s after %s %s: routing = %v, want %s", endpoint, step.action, step.path, got, step.status)
			}
			dev, _ := got["developer"].(map[string]any)
			if step.model != "" && dev["model"] != step.model {
				t.Errorf("%s selected-repository model = %v, want %s", endpoint, dev, step.model)
			}
		}
	}
}
