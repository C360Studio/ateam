package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/specs/harness/spec.md#Requirement: Committed team settings file
func TestForgesAPIRefusesCommittedList(t *testing.T) {
	for _, c := range []struct {
		name, settings string
		code           int
		body           string
	}{
		{name: "no file", code: http.StatusOK},
		{name: "file without forges", settings: `{"version":1,"merge_method":"merge"}`, code: http.StatusOK},
		{name: "committed forges", settings: `{"version":1,"forges":[{"provider":"github","owner":"team","repo":"app"}]}`,
			code: http.StatusConflict, body: "forges are committed in .agents/ateam.json"},
		{name: "invalid file", settings: `{"version":1,"forges":"github"}`, code: http.StatusConflict, body: ".agents/ateam.json"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if c.settings != "" {
				if err := os.MkdirAll(filepath.Join(dir, ".agents"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".agents", "ateam.json"), []byte(c.settings), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			srv, err := NewServer(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, payload := range []string{
				`{"provider":"gitea","base_url":"https://git.example","owner":"o","repo":"r"}`,
				`{"action":"remove","provider":"github"}`,
			} {
				req := httptest.NewRequest(http.MethodPost, "/api/forges", strings.NewReader(payload))
				req.RemoteAddr = "127.0.0.1:12345"
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
				if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.body) {
					t.Fatalf("POST %s: status %d body %s, want %d containing %q", payload, rec.Code, rec.Body.String(), c.code, c.body)
				}
			}
			if c.code == http.StatusOK {
				return
			}
			st, err := state.Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if st.ForgesConfigured || len(st.Forges) != 0 {
				t.Fatalf("a refused edit reached the cache: configured=%v forges=%+v", st.ForgesConfigured, st.Forges)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Committed team settings file
func TestForgesAPIEditAfterCommittedListRemoved(t *testing.T) {
	t.Setenv("ATEAM_FORGE", "")
	t.Setenv("GITEA_URL", "")
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "https://github.com/detected/repo.git"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	settings := filepath.Join(dir, ".agents", "ateam.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"version":1,"forges":[{"provider":"github","owner":"committed","repo":"app"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := state.Refresh(context.Background(), dir, false); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(settings); err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/forges", strings.NewReader(`{"provider":"github","owner":"newboard","repo":"app"}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST: status %d body %s, want 200", rec.Code, rec.Body.String())
	}
	st, err := state.Refresh(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if st.Forge.Owner != "newboard" || !st.ForgesConfigured || st.ForgesFromSettings {
		t.Fatalf("accepted board edit lost on refresh: forge=%+v configured=%v from settings=%v", st.Forge, st.ForgesConfigured, st.ForgesFromSettings)
	}
}
