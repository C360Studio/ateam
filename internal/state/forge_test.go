package state

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
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
					"comments": 1,
					"labels":   []map[string]any{{"name": "status:needs-decision"}},
				},
				{
					"number":   12,
					"title":    "Zero comment Gitea issue",
					"state":    "open",
					"html_url": "http://gitea.local/c360/ateam/issues/12",
					"comments": 0,
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/c360/ateam/issues/10/comments":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"body": "Decision: use Gitea v1 issues and comments endpoints",
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
	if len(cards) != 3 {
		t.Fatalf("got %d cards, want 3", len(cards))
	}
	if cards[0].Column != "needs-decision" || cards[0].Forge != "gitea" || cards[0].Decision != "Decision: use Gitea v1 issues and comments endpoints" {
		t.Errorf("unexpected issue card: %+v", cards[0])
	}
	if cards[2].Column != "in-progress" || cards[2].Agent != "codex" {
		t.Errorf("unexpected PR card: %+v", cards[2])
	}

	labels, err := BootstrapForgeLabels(context.Background(), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("BootstrapForgeLabels Gitea: %v", err)
	}
	if len(labels) != len(StandardLabels) || len(createdLabels) != len(StandardLabels) {
		t.Errorf("bootstrapped %d labels, want %d", len(labels), len(StandardLabels))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestGitHubSyncPartialFailureAndLocalCardPreservation(t *testing.T) {
	binDir := t.TempDir()
	fakeGH := filepath.Join(binDir, "gh")

	// Probe 4a: Fake gh succeeds on issue list but fails on pr list -> SyncForgeBoard must return an error.
	scriptPartial := "#!/bin/sh\nif [ \"$1\" = \"issue\" ]; then\n  echo '[{\"number\":1,\"title\":\"Issue 1\",\"state\":\"OPEN\"}]'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(fakeGH, []byte(scriptPartial), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub})
	if err == nil {
		t.Fatal("expected partial gh failure (issues ok, prs fail) to return an error, got nil")
	}

	// Probe 4b & G-06: When comments expansion fails/times out, SyncForgeBoard falls back to comment-free issue list and returns cards.
	scriptFallback := "#!/bin/sh\ncase \"$*\" in\n  *comments*)\n    exit 1\n    ;;\nesac\nif [ \"$1\" = \"issue\" ]; then\n  echo '[{\"number\":99,\"title\":\"Remote Issue\",\"state\":\"OPEN\"}]'\n  exit 0\nfi\necho '[{\"number\":100,\"title\":\"Remote PR\",\"state\":\"OPEN\",\"isDraft\":true,\"headRefName\":\"claude/fix\"}]'\n"
	if err := os.WriteFile(fakeGH, []byte(scriptFallback), 0o755); err != nil {
		t.Fatal(err)
	}
	synced, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub})
	if err != nil {
		t.Fatalf("SyncForgeBoard fallback fake gh: %v", err)
	}
	st := &State{
		Cards: []BoardCard{
			{ID: "LOCAL-1", Title: "Local task card", Column: "backlog", Forge: "local"},
		},
	}
	// Astra #2: Dashboard-created card (UpsertCard with Forge: "github") gets ATEAM-* ID and must survive empty and non-empty GitHub sync.
	uiCard, err := st.UpsertCard(BoardCard{Title: "Created in dashboard", Forge: "github", Column: "backlog"})
	if err != nil {
		t.Fatalf("UpsertCard: %v", err)
	}
	st.MergeForgeCards(nil, ForgeGitHub)
	if len(st.Cards) != 2 {
		t.Fatalf("expected 2 local/dashboard cards after empty GitHub sync, got %d: %+v", len(st.Cards), st.Cards)
	}
	st.MergeForgeCards(synced, ForgeGitHub)
	if len(st.Cards) != 4 {
		t.Fatalf("expected 4 cards after MergeForgeCards (1 local + 1 dashboard ATEAM + 2 github), got %d: %+v", len(st.Cards), st.Cards)
	}
	foundUI := false
	for _, c := range st.Cards {
		if c.ID == uiCard.ID {
			foundUI = true
		}
	}
	if !foundUI {
		t.Fatalf("dashboard card %s lost after MergeForgeCards", uiCard.ID)
	}
}
