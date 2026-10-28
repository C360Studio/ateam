package state

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

	// Probe 4c (Review Finding #2): a comments attempt that never answers is cut off at its sub-timeout, and the
	// comment-free fallback and PR lists still succeed. The attempt gets at most ghCommentTimeout under the 30s parent
	// deadline, which keeps a slow start under load far inside the margin, while an unbounded attempt hangs until the
	// deadline and fails (#164).
	scriptSlowComments := "#!/bin/sh\ncase \"$*\" in\n  *comments*)\n    exec sleep 60\n    ;;\nesac\nif [ \"$1\" = \"issue\" ]; then\n  echo '[{\"number\":99,\"title\":\"Remote Issue\",\"state\":\"OPEN\"}]'\n  exit 0\nfi\necho '[{\"number\":100,\"title\":\"Remote PR\",\"state\":\"OPEN\",\"isDraft\":true,\"headRefName\":\"claude/fix\"}]'\n"
	if err := os.WriteFile(fakeGH, []byte(scriptSlowComments), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	slow, err := SyncForgeBoard(ctx, t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"})
	if err != nil {
		t.Fatalf("SyncForgeBoard with a comments attempt that never answers: %v", err)
	}
	if len(slow) != 2 {
		t.Fatalf("expected the comment-free fallback and PR lists to yield 2 cards, got %+v", slow)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestDualForgeSyncTimestampIsolation(t *testing.T) {
	binDir := t.TempDir()
	fakeGH := filepath.Join(binDir, "gh")
	writeFake(t, fakeGH, "if [ \"$1\" = \"issue\" ]; then\n  echo '[{\"number\":99,\"title\":\"Remote Issue\",\"state\":\"OPEN\"}]'\n  exit 0\nfi\necho '[{\"number\":100,\"title\":\"Remote PR\",\"state\":\"OPEN\",\"isDraft\":true,\"headRefName\":\"claude/fix\"}]'\n")
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

// Both forge fakes hold open PRs 1-60 and closed (merged) PRs 60-130, newest first. PR 60 is in both lists: it merged
// between the open and closed reads. Sorted by last update, PR 60 heads the closed list; sorted by creation it is last,
// outside the 50-item closed cap.
const fakeOpenPRs, fakeClosedLow, fakeClosedHigh = 60, 60, 130

// fakeClosedPRNumbers lists the closed PRs in the order a forge serves them.
func fakeClosedPRNumbers(byUpdate bool) []int {
	var nums []int
	if byUpdate {
		nums = append(nums, fakeClosedLow)
	}
	for n := fakeClosedHigh; n > fakeClosedLow; n-- {
		nums = append(nums, n)
	}
	if !byUpdate {
		nums = append(nums, fakeClosedLow)
	}
	return nums
}

// assertSyncedPRs merges synced into a fresh state and checks every open PR survived, wantClosed closed PRs were kept, and PR 60's closed copy won.
func assertSyncedPRs(t *testing.T, synced []BoardCard, provider ForgeProvider, wantClosed int) {
	t.Helper()
	prs := 0
	for _, c := range synced {
		if c.Type == "pr" {
			prs++
		}
	}
	// Open PRs 1-59 plus the most recently updated closed PRs (60, then 130 down).
	if want := fakeOpenPRs - 1 + wantClosed; prs != want {
		t.Fatalf("got %d PR cards, want %d", prs, want)
	}
	st := defaultState()
	st.Git.Branch = "main"
	st.Git.Worktrees = []Worktree{
		{Path: "/repo/.worktrees/old", Branch: "claude/pr-1", Head: "abc1234"},
		{Path: "/repo/.worktrees/merged", Branch: "claude/pr-60", Head: "def5678"},
	}
	st.MergeForgeCards(synced, provider)
	for i, want := range []string{"open PR #1", "merged via #60"} {
		if got := st.WorktreePRStatus(st.Git.Worktrees[i]); got != want {
			t.Fatalf("WorktreePRStatus(%s) = %q, want %q", st.Git.Worktrees[i].Branch, got, want)
		}
	}
}

// writeGitHubFake installs a fake gh that serves PR lists from files and honours --limit like gh does. Closed lists
// come back sorted by last update only when the call asks for it with --search sort:updated-desc. With rendezvous set,
// the open-issue attempt with comments fails at once, and each of the four comment-free list calls records that it
// started and then waits until all four have started before answering. The effective bound is ghList's 5s per-call
// timeout, which kills a call still alone (the fake's own ~10s give-up never fires). Calls made one after another
// therefore fail on their own, and four concurrent calls only need to start within 5s of each other (#164).
func writeGitHubFake(t *testing.T, rendezvous bool) {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, nums []int, state string) {
		var lines []string
		for _, n := range nums {
			line := fmt.Sprintf(`{"number":%d,"title":"PR %d","state":"%s","headRefName":"claude/pr-%d"`, n, n, state, n)
			if state == "MERGED" {
				line += `,"mergedAt":"2026-10-24T00:00:00Z"`
			}
			lines = append(lines, line+"}")
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var open []int
	for n := fakeOpenPRs; n >= 1; n-- {
		open = append(open, n)
	}
	write("pr-open", open, "OPEN")
	write("pr-closed", fakeClosedPRNumbers(false), "MERGED")
	write("pr-closed-updated", fakeClosedPRNumbers(true), "MERGED")
	// argv: gh <kind> list --state <state> --limit <n> ...
	script := ""
	if rendezvous {
		started := filepath.Join(dir, "started")
		if err := os.Mkdir(started, 0o755); err != nil {
			t.Fatal(err)
		}
		script += "case \"$*\" in *comments*) exit 1;; esac\n" +
			"s='" + started + "'\n" +
			": > \"$s/$1-$4\"\n" +
			"i=0\n" +
			"while [ \"$(ls \"$s\" | wc -l)\" -lt 4 ]; do\n" +
			"  i=$((i+1)); [ \"$i\" -gt 200 ] && exit 1\n" +
			"  sleep 0.05\n" +
			"done\n"
	}
	script += "f='" + dir + "'/$1-$4\n" +
		"case \"$*\" in *sort:updated-desc*) f=\"$f-updated\";; esac\n" +
		"if [ ! -f \"$f\" ]; then echo '[]'; exit 0; fi\n" +
		"printf '['; head -n \"$6\" \"$f\" | paste -sd, -; printf ']'\n"
	writeFake(t, filepath.Join(dir, "gh"), script)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// writeFake writes an executable /bin/sh fake with body after the shebang and runs it once, outside any deadline, with
// ATEAM_FAKE_PREWARM set so it exits before doing anything. macOS checks a new executable on its first exec; under
// the full -race suite that check took up to 2.7s per file (#232, #233), enough to trip a sync deadline. Later execs
// of the same file skip it, so the test's timed calls measure the code under test.
func writeFake(t *testing.T, path, body string) {
	t.Helper()
	script := "#!/bin/sh\n[ -n \"$ATEAM_FAKE_PREWARM\" ] && exit 0\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), "ATEAM_FAKE_PREWARM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prewarm %s: %v\n%s", path, err, out)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// Scenario "Open work is never capped" (openspec/changes/forge-sync-cap-149) joins the main spec on archive.
func TestGitHubSyncNeverCapsOpenPRs(t *testing.T) {
	writeGitHubFake(t, false)
	synced, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub})
	if err != nil {
		t.Fatalf("SyncForgeBoard: %v", err)
	}
	assertSyncedPRs(t, synced, ForgeGitHub, closedItemCap)
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// Scenario "Open work is never capped" (openspec/changes/forge-sync-cap-149) joins the main spec on archive.
func TestGitHubSyncRunsListsConcurrently(t *testing.T) {
	// The sync fits the 3s pickup budget because it costs the slowest list call, not the sum. The rendezvous fake proves
	// the four list calls are in flight together: each answers only once all four have started.
	writeGitHubFake(t, true)
	synced, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub})
	if err != nil {
		t.Fatalf("SyncForgeBoard with list calls that answer only once all four have started: %v", err)
	}
	assertSyncedPRs(t, synced, ForgeGitHub, closedItemCap)
}

// newGiteaFake serves open PRs 1-60, closed PRs 60-130, and open issues 1-55, newest first. A closed PR list is sorted
// by last update only when asked with sort=recentupdate. clamp > 0 caps the page size the way [api] MAX_RESPONSE_ITEMS
// does; ignorePage serves page 1 for every page.
func newGiteaFake(t *testing.T, clamp int, ignorePage bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		isPR := strings.HasSuffix(r.URL.Path, "/pulls")
		var nums []int
		switch {
		case isPR && q.Get("state") == "open":
			for n := fakeOpenPRs; n >= 1; n-- {
				nums = append(nums, n)
			}
		case isPR && q.Get("state") == "closed":
			nums = fakeClosedPRNumbers(q.Get("sort") == "recentupdate")
		case !isPR && q.Get("type") == "issues" && q.Get("state") == "open":
			for n := 55; n >= 1; n-- {
				nums = append(nums, n)
			}
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit < 1 {
			limit = 50
		}
		if clamp > 0 && limit > clamp {
			limit = clamp
		}
		page, _ := strconv.Atoi(q.Get("page"))
		if page < 1 || ignorePage {
			page = 1
		}
		items := []map[string]any{}
		for i := (page - 1) * limit; i < len(nums) && i < page*limit; i++ {
			n := nums[i]
			item := map[string]any{"number": n, "title": fmt.Sprintf("item %d", n), "state": q.Get("state")}
			if isPR {
				item["head"] = map[string]any{"ref": fmt.Sprintf("claude/pr-%d", n)}
				item["merged"] = q.Get("state") == "closed"
			}
			items = append(items, item)
		}
		_ = json.NewEncoder(w).Encode(items)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// Scenario "Open work is never capped" (openspec/changes/forge-sync-cap-149) joins the main spec on archive.
func TestGiteaSyncNeverCapsOpenPRs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		clamp      int
		wantClosed int
	}{
		{name: "server honours limit", clamp: 0, wantClosed: closedItemCap},
		// The clamp also shortens the single closed page; open work must still page through all 60.
		{name: "server clamps page size to 30", clamp: 30, wantClosed: 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newGiteaFake(t, tc.clamp, false)
			synced, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "c360", Repo: "ateam"})
			if err != nil {
				t.Fatalf("SyncForgeBoard: %v", err)
			}
			issues := 0
			for _, c := range synced {
				if c.Type == "issue" {
					issues++
				}
			}
			if issues != 55 {
				t.Fatalf("got %d issue cards, want 55", issues)
			}
			assertSyncedPRs(t, synced, ForgeGitea, tc.wantClosed)
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// Scenario "Open work is never capped" (openspec/changes/forge-sync-cap-149) joins the main spec on archive.
func TestGiteaSyncStopsWhenServerIgnoresPage(t *testing.T) {
	srv := newGiteaFake(t, 0, true)
	// Without the no-new-numbers stop the paging loop only ends when this deadline kills it.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	synced, err := SyncForgeBoard(ctx, t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "c360", Repo: "ateam"})
	if err != nil {
		t.Fatalf("SyncForgeBoard: %v", err)
	}
	if len(synced) == 0 {
		t.Fatal("got no cards, want page 1 of each list")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestGitHubCommentBudgetIsHalfTheParentDeadline(t *testing.T) {
	if got := ghCommentBudget(context.Background()); got != ghCommentTimeout {
		t.Fatalf("budget with no deadline = %v, want ghCommentTimeout %v", got, ghCommentTimeout)
	}
	// The 3s pickup budget must leave the comments call about 1.5s: it takes about 0.8s on a real repository (#162).
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if got := ghCommentBudget(ctx); got < 1400*time.Millisecond || got > 1500*time.Millisecond {
		t.Fatalf("budget under a 3s deadline = %v, want about 1.5s", got)
	}
	// A long deadline must not stretch a hung call: the UI syncs under 10s.
	long, cancelLong := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLong()
	if got := ghCommentBudget(long); got != ghCommentTimeout {
		t.Fatalf("budget under a 10s deadline = %v, want the ghCommentTimeout cap %v", got, ghCommentTimeout)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestGitHubSyncGivesSlowCommentsTheDeadlineBudget(t *testing.T) {
	// A comments call slower than the old fixed 750ms cap (a real repository takes about 0.8s) must land rather than
	// fall back without comment bodies (#162). The call takes 1s against a 10s budget, so load cannot flip the result;
	// TestGitHubCommentBudgetIsHalfTheParentDeadline owns the budget arithmetic.
	saved := ghCommentTimeout
	ghCommentTimeout = 10 * time.Second
	t.Cleanup(func() { ghCommentTimeout = saved })
	installForgeCLI(t, "gh", "case \"$*\" in *comments*) sleep 1; printf '[{\"number\":7,\"title\":\"Pick a store\",\"state\":\"OPEN\",\"comments\":[{\"body\":\"Decision: X\"}]}]'; exit 0;; esac\nprintf '[]'\n")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cards, err := SyncForgeBoard(ctx, t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "chosen", Repo: "target"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].Decision != "Decision: X" {
		t.Fatalf("cards = %+v, want issue #7 with Decision from its slow comments call", cards)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestGiteaSyncFetchesCommentsConcurrentlyWithinLimit(t *testing.T) {
	const issues = 20
	var (
		mu       sync.Mutex
		inFlight int
		peak     int
		released = make(chan struct{})
		release  sync.Once
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var items []map[string]any
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments"):
			// Hold every fetch until the limit is in flight, then give an unbounded caller a moment to exceed it before
			// letting every fetch through. Fetches made one at a time never reach the limit and are let through by the
			// 1s timer instead, inside fetchGiteaIssueComments' 1.5s timeout.
			mu.Lock()
			inFlight++
			peak = max(peak, inFlight)
			switch {
			case inFlight > giteaCommentFetchLimit:
				release.Do(func() { close(released) })
			case inFlight == giteaCommentFetchLimit:
				time.AfterFunc(250*time.Millisecond, func() { release.Do(func() { close(released) }) })
			}
			mu.Unlock()
			select {
			case <-released:
			case <-time.After(time.Second):
				release.Do(func() { close(released) })
			}
			mu.Lock()
			inFlight--
			mu.Unlock()
			n := path.Base(path.Dir(r.URL.Path))
			_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "Decision: issue " + n}})
			return
		case q.Get("type") == "issues" && q.Get("state") == "open" && q.Get("page") == "1":
			for n := issues; n >= 1; n-- {
				items = append(items, map[string]any{"number": n, "title": fmt.Sprintf("issue %d", n), "state": "open", "comments": 1})
			}
		}
		if items == nil {
			items = []map[string]any{}
		}
		_ = json.NewEncoder(w).Encode(items)
	}))
	t.Cleanup(srv.Close)

	cards, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "c360", Repo: "ateam"})
	if err != nil {
		t.Fatalf("SyncForgeBoard: %v", err)
	}
	mu.Lock()
	gotPeak := peak
	mu.Unlock()
	if gotPeak != giteaCommentFetchLimit {
		t.Fatalf("peak concurrent comment fetches = %d, want exactly the limit %d", gotPeak, giteaCommentFetchLimit)
	}
	if len(cards) != issues {
		t.Fatalf("got %d cards, want %d", len(cards), issues)
	}
	for _, c := range cards {
		if want := fmt.Sprintf("Decision: issue %d", c.Number); c.Decision != want {
			t.Errorf("card #%d Decision = %q, want %q", c.Number, c.Decision, want)
		}
	}
}
