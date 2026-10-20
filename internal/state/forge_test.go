package state

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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
			{ID: "ATEAM-1", Number: 1, Title: "Universal CLI lifecycle hooks (Claude, Codex, Gemini) <5ms", Column: "done", Type: "epic", Agent: "gemini", Branch: "gemini/universal-hooks", Milestone: "v0.1 Foundation", Forge: "github"},
			{ID: "GH-404", Number: 404, Title: "Deleted GitHub issue", Column: "backlog", Forge: "github"},
			{ID: "GT-7", Number: 7, Title: "Gitea issue", Column: "in-progress", Forge: "gitea"},
			{ID: "GH-99", Number: 99, Title: "Remote Issue", Column: "backlog", Forge: "github", HoldReason: "Hold: waiting on schema", Decision: "Decision: keep hold"},
		},
	}
	uiCard, err := st.UpsertCard(BoardCard{Title: "Real dashboard task", Forge: "github", Column: "backlog", HoldReason: "Hold: dashboard hold"})
	if err != nil {
		t.Fatalf("UpsertCard: %v", err)
	}
	demoTitleTask, err := st.UpsertCard(BoardCard{Title: "Universal CLI lifecycle hooks (Claude, Codex, Gemini) <5ms", Column: "backlog", HoldReason: "Hold: user hold"})
	if err != nil {
		t.Fatalf("UpsertCard demoTitleTask: %v", err)
	}
	demoBranchTask, err := st.UpsertCard(BoardCard{Title: "Follow-up on universal hooks", Branch: "gemini/universal-hooks", Column: "in-progress"})
	if err != nil {
		t.Fatalf("UpsertCard demoBranchTask: %v", err)
	}
	cardSaveDir := t.TempDir()
	if err := Save(cardSaveDir, st); err != nil {
		t.Fatalf("Save: %v", err)
	}
	st, err = Load(cardSaveDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	st.MergeForgeCards(synced, ForgeGitHub)
	if st.BoardSyncedAt.IsZero() {
		t.Fatal("expected BoardSyncedAt to be recorded after MergeForgeCards")
	}
	byID := make(map[string]BoardCard, len(st.Cards))
	for _, c := range st.Cards {
		byID[c.ID] = c
	}
	if _, ok := byID["ATEAM-1"]; ok {
		t.Fatalf("legacy demo card ATEAM-1 survived Load/MergeForgeCards: %+v", st.Cards)
	}
	if _, ok := byID["GH-404"]; ok {
		t.Fatalf("stale github card GH-404 survived MergeForgeCards: %+v", st.Cards)
	}
	if gotUI, ok := byID[uiCard.ID]; !ok || gotUI.HoldReason != "Hold: dashboard hold" {
		t.Fatalf("legitimate UpsertCard %s or its HoldReason was lost: %+v", uiCard.ID, st.Cards)
	}
	if gotTitle, ok := byID[demoTitleTask.ID]; !ok || gotTitle.HoldReason != "Hold: user hold" {
		t.Fatalf("legitimate UpsertCard with demo title %s was deleted: %+v", demoTitleTask.ID, st.Cards)
	}
	if gotBranch, ok := byID[demoBranchTask.ID]; !ok || gotBranch.Branch != "gemini/universal-hooks" {
		t.Fatalf("legitimate UpsertCard with demo branch %s was deleted: %+v", demoBranchTask.ID, st.Cards)
	}
	if _, ok := byID["GT-7"]; !ok {
		t.Fatalf("other-forge card GT-7 was deleted by GitHub sync: %+v", st.Cards)
	}
	if got99, ok := byID["GH-99"]; !ok || got99.HoldReason != "Hold: waiting on schema" || got99.Decision != "Decision: keep hold" {
		t.Fatalf("matching card GH-99 lost HoldReason or Decision: %+v", got99)
	}

	// Probe 4c (Review Finding #2): Fake gh that blocks on requests containing `comments` and returns instantly otherwise
	// must NOT exhaust RefreshWithBestEffortSync's 3s parent timeout; comment-free fallback + PR query must succeed and advance BoardSyncedAt.
	scriptSlowComments := "#!/bin/sh\ncase \"$*\" in\n  *comments*)\n    sleep 5\n    exit 1\n    ;;\nesac\nif [ \"$1\" = \"issue\" ]; then\n  echo '[{\"number\":99,\"title\":\"Remote Issue\",\"state\":\"OPEN\"}]'\n  exit 0\nfi\necho '[{\"number\":100,\"title\":\"Remote PR\",\"state\":\"OPEN\",\"isDraft\":true,\"headRefName\":\"claude/fix\"}]'\n"
	if err := os.WriteFile(fakeGH, []byte(scriptSlowComments), 0o755); err != nil {
		t.Fatal(err)
	}
	syncRoot := t.TempDir()
	if _, err := Update(syncRoot, func(cur *State) error {
		return cur.UpsertForge(ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"})
	}); err != nil {
		t.Fatalf("UpsertForge: %v", err)
	}
	refreshed, err := RefreshWithBestEffortSync(context.Background(), syncRoot, 0)
	if err != nil {
		t.Fatalf("RefreshWithBestEffortSync slow comments: %v", err)
	}
	if refreshed.BoardSyncedAt.IsZero() || refreshed.BoardSyncError != "" || len(refreshed.Cards) != 2 {
		t.Fatalf("expected slow comments to fall back and advance BoardSyncedAt, got syncedAt=%v err=%q cards=%+v",
			refreshed.BoardSyncedAt, refreshed.BoardSyncError, refreshed.Cards)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestDualForgeSyncTimestampIsolation(t *testing.T) {
	binDir := t.TempDir()
	fakeGH := filepath.Join(binDir, "gh")
	scriptGH := "#!/bin/sh\nif [ \"$1\" = \"issue\" ]; then\n  echo '[{\"number\":99,\"title\":\"Remote Issue\",\"state\":\"OPEN\"}]'\n  exit 0\nfi\necho '[{\"number\":100,\"title\":\"Remote PR\",\"state\":\"OPEN\",\"isDraft\":true,\"headRefName\":\"claude/fix\"}]'\n"
	if err := os.WriteFile(fakeGH, []byte(scriptGH), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dualRoot := t.TempDir()
	oldGHSync := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if _, err := Update(dualRoot, func(cur *State) error {
		if err := cur.UpsertForge(ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"}); err != nil {
			return err
		}
		cur.BoardSyncedAt = oldGHSync
		cur.Cards = []BoardCard{{ID: "GH-1", Number: 1, Title: "Stale GitHub card", Column: "backlog", Forge: "github"}}
		cur.MergeForgeCards([]BoardCard{{ID: "GT-1", Number: 1, Title: "Gitea item", Column: "backlog", Forge: "gitea"}}, ForgeGitea)
		return nil
	}); err != nil {
		t.Fatalf("Update dualRoot: %v", err)
	}
	afterGiteaMerge, err := Load(dualRoot)
	if err != nil {
		t.Fatalf("Load dualRoot: %v", err)
	}
	if !afterGiteaMerge.BoardSyncedAt.Equal(oldGHSync) {
		t.Fatalf("BoardSyncedAt after Gitea merge = %v, want %v", afterGiteaMerge.BoardSyncedAt, oldGHSync)
	}
	if snap := afterGiteaMerge.BoardView(); snap.Forge != ForgeGitHub || snap.SyncedAt != oldGHSync.Format(time.RFC3339) {
		t.Fatalf("BoardView after Gitea merge reported (%s, %s), want (github, %s)", snap.Forge, snap.SyncedAt, oldGHSync.Format(time.RFC3339))
	}
	dualRefreshed, err := RefreshWithBestEffortSync(context.Background(), dualRoot, 30*time.Second)
	if err != nil {
		t.Fatalf("RefreshWithBestEffortSync dualRoot: %v", err)
	}
	if !dualRefreshed.SyncedAtFor(dualRefreshed.Forge).After(oldGHSync) {
		t.Fatalf("expected RefreshWithBestEffortSync to sync stale GitHub board despite recent Gitea merge, got %v", dualRefreshed.SyncedAtFor(dualRefreshed.Forge))
	}
	dualByID := make(map[string]BoardCard, len(dualRefreshed.Cards))
	for _, c := range dualRefreshed.Cards {
		dualByID[c.ID] = c
	}
	if _, ok := dualByID["GH-1"]; ok {
		t.Fatalf("stale GH-1 card survived RefreshWithBestEffortSync after Gitea merge: %+v", dualRefreshed.Cards)
	}
	if _, ok := dualByID["GH-99"]; !ok {
		t.Fatalf("expected fresh GH-99 card from GitHub sync, got: %+v", dualRefreshed.Cards)
	}
	if _, ok := dualByID["GT-1"]; !ok {
		t.Fatalf("expected Gitea card GT-1 to remain preserved, got: %+v", dualRefreshed.Cards)
	}
}
