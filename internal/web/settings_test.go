package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
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
