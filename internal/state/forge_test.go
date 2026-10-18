package state

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForgeGitHubAndGitea(t *testing.T) {
	host, owner, repo := ParseGitRemote("ssh://git@github.com/C360Studio/ateam.git")
	if host != "github.com" || owner != "C360Studio" || repo != "ateam" {
		t.Fatalf("ParseGitRemote SSH = (%q, %q, %q)", host, owner, repo)
	}

	host, owner, repo = ParseGitRemote("https://gitea.c360.local/c360/ateam.git")
	if host != "gitea.c360.local" || owner != "c360" || repo != "ateam" {
		t.Fatalf("ParseGitRemote HTTPS = (%q, %q, %q)", host, owner, repo)
	}

	var createdLabels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/c360/ateam/issues":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number":   10,
					"title":    "Support Gitea API v1",
					"state":    "open",
					"html_url": "http://gitea.local/c360/ateam/issues/10",
					"labels":   []map[string]any{{"name": "status:needs-decision"}},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/c360/ateam/pulls":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number":   11,
					"title":    "Implement Gitea sync",
					"state":    "open",
					"draft":    true,
					"html_url": "http://gitea.local/c360/ateam/pulls/11",
					"head":     map[string]any{"ref": "codex/gitea-sync"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/c360/ateam/labels":
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			createdLabels = append(createdLabels, body.Name)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := ForgeConfig{
		Provider: ForgeGitea,
		BaseURL:  srv.URL,
		Owner:    "c360",
		Repo:     "ateam",
	}
	cards, err := SyncForgeBoard(context.Background(), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("SyncForgeBoard Gitea: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("got %d cards, want 2", len(cards))
	}
	if cards[0].Column != "needs-decision" || cards[0].Forge != "gitea" {
		t.Errorf("unexpected issue card: %+v", cards[0])
	}
	if cards[1].Column != "in-progress" || cards[1].Agent != "codex" {
		t.Errorf("unexpected PR card: %+v", cards[1])
	}

	labels, err := BootstrapForgeLabels(context.Background(), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("BootstrapForgeLabels Gitea: %v", err)
	}
	if len(labels) != len(StandardLabels) || len(createdLabels) != len(StandardLabels) {
		t.Errorf("bootstrapped %d labels, want %d", len(labels), len(StandardLabels))
	}
}
