package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestHandoffPickupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load empty dir: %v", err)
	}
	if st.Mode != ModeFull {
		t.Fatalf("default mode = %q, want %q", st.Mode, ModeFull)
	}
	if len(st.Cards) != 0 {
		t.Fatalf("fresh state Cards count = %d, want 0 (no fabricated board cards)", len(st.Cards))
	}

	h := Handoff{
		Goal:          "PR #12: implement universal hooks and Gitea sync",
		Addresses:     "branch gemini/hooks, SHA abc1234",
		Constraints:   "Hold: none; write ownership transferred to codex",
		DoneWhen:      "task verify passes on clean tree",
		ChecksToRun:   "task test:unit && task spec:check",
		Agent:         "gemini",
		Branch:        "gemini/hooks",
		CheckpointSHA: "abc1234",
		UpdatedAt:     time.Now().UTC(),
	}
	st.LastHandoff = &h
	st.Mode = ModeUltra
	if err := Save(dir, st); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load saved: %v", err)
	}
	if loaded.Mode != ModeUltra {
		t.Fatalf("loaded mode = %q, want %q", loaded.Mode, ModeUltra)
	}
	if loaded.LastHandoff == nil || loaded.LastHandoff.Goal != h.Goal {
		t.Fatalf("loaded handoff mismatch: %+v", loaded.LastHandoff)
	}

	block := FormatFiveFieldPrompt(*loaded.LastHandoff)
	for _, want := range []string{
		"Goal: PR #12: implement universal hooks and Gitea sync",
		"Addresses: branch gemini/hooks, SHA abc1234",
		"Constraints: Hold: none; write ownership transferred to codex",
		"Done when: task verify passes on clean tree",
		"Checks to run: task test:unit && task spec:check",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("FormatFiveFieldPrompt missing %q in:\n%s", want, block)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Inline shortcut inventory
func TestScanShortcuts(t *testing.T) {
	dir := t.TempDir()
	src := `package sample

// shortcut: in-memory cache only, upgrade to NATS KV when multi-node is enabled
func compute() int {
	return 42
}
`
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write sample.go: %v", err)
	}
	items, err := ScanShortcuts(dir)
	if err != nil {
		t.Fatalf("ScanShortcuts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d shortcuts, want 1", len(items))
	}
	if items[0].File != "sample.go" || items[0].Line != 3 {
		t.Errorf("unexpected location: %+v", items[0])
	}
	if items[0].Limit != "in-memory cache only" || items[0].RevisitTrigger != "upgrade to NATS KV when multi-node is enabled" {
		t.Errorf("unexpected parsed shortcut: %+v", items[0])
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestConcurrentUpdateAndCorruptCachePreservation(t *testing.T) {
	dir := t.TempDir()

	// Probe 2: Concurrent / sequential state.Update calls must preserve both handoff and mode without clobbering.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := Update(dir, func(st *State) error {
			st.LastHandoff = &Handoff{Goal: "Preserve handoff across concurrent updates"}
			return nil
		})
		if err != nil {
			t.Errorf("Update handoff: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := Update(dir, func(st *State) error {
			st.Mode = ModeUltra
			return nil
		})
		if err != nil {
			t.Errorf("Update mode: %v", err)
		}
	}()
	wg.Wait()

	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after concurrent Update: %v", err)
	}
	if st.Mode != ModeUltra {
		t.Fatalf("st.Mode = %q, want %q", st.Mode, ModeUltra)
	}
	if st.LastHandoff == nil || st.LastHandoff.Goal != "Preserve handoff across concurrent updates" {
		t.Fatalf("LastHandoff lost after concurrent Update: %+v", st.LastHandoff)
	}

	// Probe 3: Malformed cache must not be overwritten by Refresh.
	corruptRaw := []byte(`{"mode":"ultra","last_handoff":{"goal":"partialcorrupt`)
	if err := os.WriteFile(CacheFilePath(dir), corruptRaw, 0o644); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}
	if _, err := Refresh(context.Background(), dir, false); err == nil {
		t.Fatal("expected Refresh to fail on malformed state.json, got nil")
	}
	afterBytes, err := os.ReadFile(CacheFilePath(dir))
	if err != nil {
		t.Fatalf("read cache after failed Refresh: %v", err)
	}
	if string(afterBytes) != string(corruptRaw) {
		t.Fatalf("Refresh overwrote corrupt cache: got %q, want %q", string(afterBytes), string(corruptRaw))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Fast Git identity across worktrees
func TestReadFastGitHeadLinkedWorktree(t *testing.T) {
	base := t.TempDir()
	mainGit := filepath.Join(base, "main-repo", ".git")
	wtGitDir := filepath.Join(mainGit, "worktrees", "wt-gemini")
	wtRoot := filepath.Join(base, "wt-gemini")

	if err := os.MkdirAll(filepath.Join(mainGit, "refs", "heads", "gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wtGitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wtRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Linked worktree .git file pointing to wtGitDir, and commondir pointing to ../..
	if err := os.WriteFile(filepath.Join(wtRoot, ".git"), []byte("gitdir: ../main-repo/.git/worktrees/wt-gemini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGitDir, "HEAD"), []byte("ref: refs/heads/gemini/worktree-fix\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainGit, "refs", "heads", "gemini", "worktree-fix"), []byte("deadbeefcafe1234567890\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	br, sha := ReadFastGitHead(wtRoot)
	if br != "gemini/worktree-fix" || sha != "deadbee" {
		t.Fatalf("ReadFastGitHead linked worktree loose ref = (%q, %q), want (\"gemini/worktree-fix\", \"deadbee\")", br, sha)
	}

	// Also verify packed-refs fallback via commondir.
	_ = os.Remove(filepath.Join(mainGit, "refs", "heads", "gemini", "worktree-fix"))
	packed := "# pack-refs with: peeled fully-peeled sorted\n9876543210abcdef1234567890abcdef12345678 refs/heads/gemini/worktree-fix\n"
	if err := os.WriteFile(filepath.Join(mainGit, "packed-refs"), []byte(packed), 0o644); err != nil {
		t.Fatal(err)
	}
	br, sha = ReadFastGitHead(wtRoot)
	if br != "gemini/worktree-fix" || sha != "9876543" {
		t.Fatalf("ReadFastGitHead linked worktree packed-ref = (%q, %q), want (\"gemini/worktree-fix\", \"9876543\")", br, sha)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestPostPRHandoffCommentAndDecisionsInPickup(t *testing.T) {
	var postedComment string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number": 12,
					"title":  "Hardening fixes",
					"state":  "open",
					"draft":  true,
					"head":   map[string]string{"ref": "gemini/hardening"},
				},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues/12/comments"):
			raw, _ := io.ReadAll(r.Body)
			var body struct {
				Body string `json:"body"`
			}
			_ = json.Unmarshal(raw, &body)
			postedComment = body.Body
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number": 7,
					"title":  "Auth token policy",
					"state":  "open",
					"body":   "Decision: require ATEAM_API_TOKEN for all non-loopback reads and writes",
					"labels": []map[string]string{{"name": "decision:approved"}},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Provider: ForgeGitea,
		BaseURL:  ts.URL,
		Owner:    "c360",
		Repo:     "ateam",
	}
	h := Handoff{
		Goal:        "Land PR #12 review fixes",
		Addresses:   "PR #12",
		Constraints: "none",
		DoneWhen:    "task verify green",
		ChecksToRun: "task verify",
	}
	wantPrompt := strings.TrimSpace(FormatFiveFieldPrompt(h))
	posted, err := PostPRHandoffComment(context.Background(), t.TempDir(), cfg, "gemini/hardening", wantPrompt)
	if err != nil || !posted {
		t.Fatalf("PostPRHandoffComment = (%v, %v), want (true, nil)", posted, err)
	}
	if strings.TrimSpace(postedComment) != wantPrompt {
		t.Fatalf("posted PR comment = %q, want %q", postedComment, wantPrompt)
	}

	cards, err := SyncForgeBoard(context.Background(), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("SyncForgeBoard: %v", err)
	}
	st := defaultState()
	st.MergeForgeCards(cards, ForgeGitea)
	summary := BuildPickupSummary(st)
	if !strings.Contains(summary, "Recent decisions:") || !strings.Contains(summary, "Decision: require ATEAM_API_TOKEN") {
		t.Fatalf("BuildPickupSummary missing decision summary:\n%s", summary)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional herdr pane identity in the writer lease
func TestWriterLeaseHerdrIdentity(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		wantPane      string
		wantWorkspace string
	}{
		{
			name:          "inside herdr records pane and workspace",
			env:           map[string]string{"HERDR_ENV": "1", "HERDR_PANE_ID": "p_7", "HERDR_WORKSPACE_ID": "w_2"},
			wantPane:      "p_7",
			wantWorkspace: "w_2",
		},
		{
			name: "outside herdr records nothing",
			env:  map[string]string{"HERDR_ENV": "", "HERDR_PANE_ID": "", "HERDR_WORKSPACE_ID": ""},
		},
		{
			name: "stray pane id without HERDR_ENV is ignored",
			env:  map[string]string{"HERDR_ENV": "", "HERDR_PANE_ID": "p_7", "HERDR_WORKSPACE_ID": "w_2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			dir := t.TempDir()
			if _, err := WriteWriterLease(dir, "claude", "s1"); err != nil {
				t.Fatalf("WriteWriterLease: %v", err)
			}
			raw, err := os.ReadFile(WriterLeasePath(dir))
			if err != nil {
				t.Fatal(err)
			}
			got, err := ReadWriterLease(dir)
			if err != nil {
				t.Fatalf("ReadWriterLease: %v", err)
			}
			if got.HerdrPaneID != tc.wantPane || got.HerdrWorkspaceID != tc.wantWorkspace {
				t.Fatalf("lease herdr ids = (%q, %q), want (%q, %q)", got.HerdrPaneID, got.HerdrWorkspaceID, tc.wantPane, tc.wantWorkspace)
			}
			if tc.wantPane == "" && strings.Contains(string(raw), "herdr_") {
				t.Fatalf("writer.json should omit herdr fields outside herdr, got:\n%s", raw)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional herdr pane identity in the writer lease
// spec: openspec/specs/harness/spec.md#Requirement: Lifecycle hook normalization under 5ms
func TestPickupWarningNamesHerdrPane(t *testing.T) {
	t.Setenv("ATEAM_SESSION_ID", "")
	livePID := os.Getpid()
	base := WriterLease{Agent: "codex", SessionID: "other", PID: livePID, LastSeen: time.Now().UTC()}

	withPane := base
	withPane.HerdrPaneID = "p_9"
	out := BuildPickupSummary(&State{Writer: &withPane, CurrentSessionID: "mine"})
	for _, want := range []string{
		fmt.Sprintf("Writer lease: agent=codex session=other pid=%d last_seen=", livePID),
		fmt.Sprintf(" ago (herdr pane p_9)\nWARNING: another writer may be active (agent=codex, session=other, pid=%d, last_seen=", livePID),
		" ago) (herdr pane p_9)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("pickup summary missing %q:\n%s", want, out)
		}
	}

	out = BuildPickupSummary(&State{Writer: &base, CurrentSessionID: "mine"})
	if !strings.Contains(out, "WARNING: another writer may be active") || strings.Contains(out, "herdr pane") {
		t.Fatalf("pickup summary without pane should warn but not name a pane:\n%s", out)
	}

	// #40: Dead PID must report (exited) and must NOT warn as an active competing writer.
	exitedLease := WriterLease{Agent: "claude", SessionID: "other", PID: 999999, LastSeen: time.Now().UTC()}
	exitedOut := BuildPickupSummary(&State{Writer: &exitedLease, CurrentSessionID: "mine"})
	if !strings.Contains(exitedOut, "pid=999999 (exited)") || strings.Contains(exitedOut, "WARNING: another writer may be active") {
		t.Fatalf("exited PID lease should not warn as active writer:\n%s", exitedOut)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestHandoffStaleInvalidationAndRefreshWithBestEffortSync(t *testing.T) {
	t.Setenv("ATEAM_SESSION_ID", "")
	st := &State{
		Mode: ModeFull,
		Git:  GitStatus{Branch: "main", HeadSHA: "9f11a42"},
		Cards: []BoardCard{
			{ID: "GH-PR-2", Number: 2, Title: "Merged Astra PR", Type: "pr", Column: "done", Forge: "github"},
			{ID: "GH-PR-45", Number: 45, Title: "Merged context PR", Type: "pr", Column: "done", Forge: "github"},
		},
		OpenSpecQueue: []OpenSpecChange{
			{Name: "in-flight-change", CompletedTasks: 1, TotalTasks: 2},
			{Name: "completed-change", CompletedTasks: 3, TotalTasks: 3},
		},
		LastHandoff: &Handoff{
			Goal:          "Review and merge PR #2 remediating all 8 Astra findings",
			Addresses:     "PR #2 (https://github.com/C360Studio/ateam/pull/2)",
			DoneWhen:      "PR #2 merged into main",
			ChecksToRun:   "task verify",
			Branch:        "main",
			CheckpointSHA: "02e913f",
		},
	}
	st.RefreshHandoffStaleness()
	if !st.LastHandoff.Stale {
		t.Fatalf("expected drifted/merged-PR handoff to be marked stale, got %+v", st.LastHandoff)
	}
	for _, wantReason := range []string{"HEAD drifted from 02e913f -> 9f11a42", "PR #2 is closed/merged"} {
		if !strings.Contains(st.LastHandoff.StaleReason, wantReason) {
			t.Fatalf("StaleReason = %q, missing %q", st.LastHandoff.StaleReason, wantReason)
		}
	}
	summary := BuildPickupSummary(st)
	if !strings.Contains(summary, "Last Handoff (HEAD drifted from 02e913f -> 9f11a42): STALE — handoff omitted (") {
		t.Fatalf("expected STALE handoff omitted line in summary, got:\n%s", summary)
	}
	if strings.Contains(summary, "Goal: Review and merge PR #2") {
		t.Fatalf("stale handoff Goal should not be injected in pickup summary:\n%s", summary)
	}
	if !strings.Contains(summary, "OpenSpec in-flight changes: 1\n  - in-flight-change (1/2 tasks, 0 caveats)") {
		t.Fatalf("expected active in-flight change in pickup summary, got:\n%s", summary)
	}
	if !strings.Contains(summary, "OpenSpec completed changes (done, not archived): 1\n  - completed-change (3/3 tasks, done, not archived)") {
		t.Fatalf("expected completed unarchived change in pickup summary, got:\n%s", summary)
	}

	// #53: Context PR mentions in Addresses/DoneWhen or mid-sentence in Goal must NOT mark a current-HEAD handoff stale.
	st.LastHandoff = &Handoff{
		Goal:          "Post-merge cleanup after PR #45 landed on main",
		Addresses:     "main @ 9f11a42; context: fixed earlier by PR #45",
		DoneWhen:      "PR #45 follow-up verified",
		ChecksToRun:   "task verify",
		Branch:        "main",
		CheckpointSHA: "9f11a42",
	}
	st.RefreshHandoffStaleness()
	if st.LastHandoff.Stale || len(st.HandoffStaleReasons()) != 0 {
		t.Fatalf("context PR mention falsely marked handoff stale: %+v", st.LastHandoff)
	}

	// #53: Same-session handoff at current HEAD and branch is fresh even if Goal starts with a merged PR reference.
	st.CurrentSessionID = "sess-active"
	st.LastHandoff = &Handoff{
		Goal:          "PR #45: verify post-merge state on main",
		Addresses:     "main @ 9f11a42",
		DoneWhen:      "verified",
		ChecksToRun:   "task verify",
		SessionID:     "sess-active",
		Branch:        "main",
		CheckpointSHA: "9f11a42",
	}
	st.RefreshHandoffStaleness()
	if st.LastHandoff.Stale {
		t.Fatalf("same-session current-HEAD handoff should not be marked stale: %+v", st.LastHandoff)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestResolveGitUser(t *testing.T) {
	cases := []struct {
		gitName   string
		gitEmail  string
		osUser    string
		wantName  string
		wantEmail string
	}{
		{"Coby B", "coby@c360.dev", "coby", "Coby B", "coby@c360.dev"},
		{"", "alice@example.com", "local-alice", "alice", "alice@example.com"},
		{"", "", "coby", "coby", ""},
		{"  ", "  ", "  ", "Local User", ""},
	}
	for _, tc := range cases {
		gotName, gotEmail := resolveGitUser(tc.gitName, tc.gitEmail, tc.osUser)
		if gotName != tc.wantName || gotEmail != tc.wantEmail {
			t.Fatalf("resolveGitUser(%q, %q, %q) = (%q, %q), want (%q, %q)",
				tc.gitName, tc.gitEmail, tc.osUser, gotName, gotEmail, tc.wantName, tc.wantEmail)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestParseSpecMarkdownAndScanOpenSpecSpecs(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "openspec", "specs", "harness")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `# harness Specification

## Purpose

Defines the behavioral contract for ateam hooks and MCP budget.

## Requirements

### Requirement: Lifecycle hook normalization under 5ms

ateam hook SHALL normalize events in under 5ms.

#### Scenario: Mode switch

- **WHEN** ateam hook receives /mode ultra
- **THEN** it updates state.json to ultra
`
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := ScanOpenSpecSpecs(dir)
	if err != nil {
		t.Fatalf("ScanOpenSpecSpecs: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("len(specs) = %d, want 1", len(specs))
	}
	sp := specs[0]
	if sp.Name != "harness" || !strings.Contains(sp.Purpose, "behavioral contract") {
		t.Fatalf("unexpected spec header: %+v", sp)
	}
	if len(sp.Requirements) != 1 {
		t.Fatalf("len(Requirements) = %d, want 1", len(sp.Requirements))
	}
	req := sp.Requirements[0]
	if req.ID != "harness:1" || req.Name != "Lifecycle hook normalization under 5ms" {
		t.Fatalf("unexpected requirement: %+v", req)
	}
	if len(req.Scenarios) != 1 || req.Scenarios[0].Name != "Mode switch" || len(req.Scenarios[0].Steps) != 2 {
		t.Fatalf("unexpected scenarios: %+v", req.Scenarios)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestReconcileWorktreesAndPickupWorktreeSection(t *testing.T) {
	dirtyFalse := false
	dirtyTrue := true
	st := defaultState()
	st.Git.Branch = "main"
	st.Git.HeadSHA = "e37563b"
	st.Git.Worktrees = []Worktree{
		{Path: "/repo/.worktrees/wt-57", Branch: "gemini/mvp-hardening-52", Head: "aa11bb2", Dirty: &dirtyFalse, AheadCount: 2, MergedIntoMain: false},
		{Path: "/repo/.worktrees/wt-64", Branch: "gemini/pickup-mcp-62-63", Head: "cc33dd4", Dirty: &dirtyTrue, AheadCount: 1, MergedIntoMain: false},
		{Path: "/repo/.worktrees/wt-55", Branch: "codex/follow-up", Head: "ff99ee0", Dirty: &dirtyFalse, AheadCount: 3, MergedIntoMain: true},
		{Path: "/repo/.worktrees/wt-56", Branch: "claude/abandoned", Head: "1122334", Dirty: &dirtyFalse, AheadCount: 1, MergedIntoMain: false},
	}
	st.Cards = []BoardCard{
		{ID: "GH-PR-57", Number: 57, Type: "pr", Column: "done", Merged: true, Branch: "gemini/mvp-hardening-52", HeadSHA: "aa11bb29999"},
		{ID: "GH-PR-64", Number: 64, Type: "pr", Column: "in_progress", Merged: false, Branch: "gemini/pickup-mcp-62-63", HeadSHA: "cc33dd40000"},
		{ID: "GH-PR-55", Number: 55, Type: "pr", Column: "done", Merged: true, Branch: "codex/follow-up", HeadSHA: "0011223344"},
		{ID: "GH-PR-56", Number: 56, Type: "pr", Column: "done", Merged: false, Branch: "claude/abandoned", HeadSHA: "1122334"},
	}

	st.ReconcileWorktrees()

	if !st.Git.Worktrees[0].MergedIntoMain {
		t.Fatalf("expected squash-merged worktree wt-57 with matching head_sha to have MergedIntoMain=true, got false")
	}
	if st.Git.Worktrees[1].MergedIntoMain {
		t.Fatalf("expected open-PR worktree wt-64 to have MergedIntoMain=false, got true")
	}
	if st.Git.Worktrees[2].MergedIntoMain {
		t.Fatalf("expected worktree wt-55 with commits ahead of merged PR head_sha to have MergedIntoMain=false, got true")
	}
	if st.Git.Worktrees[3].MergedIntoMain {
		t.Fatalf("expected closed-unmerged worktree wt-56 to have MergedIntoMain=false, got true")
	}

	summary := BuildPickupSummary(st)
	for _, want := range []string{
		"Worktrees: 4",
		"- gemini/mvp-hardening-52 (head=aa11bb2, dirty=false, ahead=2, merged via #57)",
		"- gemini/pickup-mcp-62-63 (head=cc33dd4, dirty=true, ahead=1, open PR #64)",
		"- codex/follow-up (head=ff99ee0, dirty=false, ahead=3, ahead of merged #55)",
		"- claude/abandoned (head=1122334, dirty=false, ahead=1, closed without merge (#56))",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected pickup summary to contain %q, got:\n%s", want, summary)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestPickupLabelsLiveDetachedWorktreeDetached(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "refs", "heads", "main"), []byte("abad85b000000000000000000000000000000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wtPath := filepath.Join(root, ".worktrees", "codex-review-pr-86")
	if err := os.MkdirAll(filepath.Join(wtPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, ".git", "HEAD"), []byte("a30ca686e0000000000000000000000000000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirtyFalse := false
	st := defaultState()
	st.Git.Branch = "main"
	st.Git.HeadSHA = "abad85b"
	st.Git.Worktrees = []Worktree{
		{Path: root, Branch: "main", Head: "abad85b", Dirty: &dirtyFalse, MergedIntoMain: true},
		{Path: wtPath, Branch: "", Head: "a30ca68", Dirty: &dirtyFalse, AheadCount: 4},
	}
	if err := Save(root, st); err != nil {
		t.Fatal(err)
	}

	ReconcilePickup(root, st, "claude", "sess-96")

	// Rendering also reads "HEAD" as detached, so check the stored form the live read writes.
	if got := st.Git.Worktrees[1].Branch; got != "" {
		t.Fatalf("expected live detached read to store branch %q, got %q", "", got)
	}
	summary := BuildPickupSummary(st)
	if want := "- detached (head=a30ca68, dirty=false, ahead=4, detached)"; !strings.Contains(summary, want) {
		t.Fatalf("expected pickup summary to contain %q, got:\n%s", want, summary)
	}
	if strings.Contains(summary, "- HEAD (") {
		t.Fatalf("expected no worktree named HEAD, got:\n%s", summary)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestWorktreeNoPRAndStaleForgeCacheCaveat(t *testing.T) {
	dirtyFalse := false
	st := defaultState()
	st.Forge = ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"}
	st.Git.Branch = "main"
	st.Git.HeadSHA = "abad85b"
	st.Git.Worktrees = []Worktree{
		{Path: "/repo", Branch: "main", Head: "abad85b", Dirty: &dirtyFalse, AheadCount: 0, MergedIntoMain: true},
		{Path: "/repo/.worktrees/wt-ancestor", Branch: "claude/ancestor", Head: "9a8b7c6", Dirty: &dirtyFalse, AheadCount: 0, MergedIntoMain: true},
		{Path: "/repo/.worktrees/wt-fresh", Branch: "gemini/fresh", Head: "abad85b", Dirty: &dirtyFalse, AheadCount: 0, MergedIntoMain: true},
		{Path: "/repo/.worktrees/wt-64", Branch: "codex/mercury-offline-contract", Head: "e50ca10f", Dirty: &dirtyFalse, AheadCount: 4, MergedIntoMain: false},
		{Path: "/repo/.worktrees/wt-review", Branch: "codex/review-pr-79", Head: "d7bb7968", Dirty: &dirtyFalse, AheadCount: 5, MergedIntoMain: false},
	}
	st.Cards = []BoardCard{
		{ID: "GH-PR-64", Number: 64, Type: "pr", Column: "in_progress", Merged: false, Branch: "codex/mercury-offline-contract", HeadSHA: "e50ca10f"},
	}

	// Fresh cache (<10m): no cache-age suffix or caveat line; worktree with no PR is labeled "no PR" (not "unmerged").
	st.BoardSyncedAt = time.Now().UTC().Add(-2 * time.Minute)
	freshSummary := BuildPickupSummary(st)
	for _, want := range []string{
		"- main (head=abad85b, dirty=false, ahead=0, default branch)",
		"- claude/ancestor (head=9a8b7c6, dirty=false, ahead=0, merged (ancestor of main))",
		"- gemini/fresh (head=abad85b, dirty=false, ahead=0, no commits yet)",
		"- codex/mercury-offline-contract (head=e50ca10f, dirty=false, ahead=4, open PR #64)",
		"- codex/review-pr-79 (head=d7bb7968, dirty=false, ahead=5, no PR)",
	} {
		if !strings.Contains(freshSummary, want) {
			t.Fatalf("expected fresh-cache pickup summary to contain %q, got:\n%s", want, freshSummary)
		}
	}
	if strings.Contains(freshSummary, "labels from forge cache synced") || strings.Contains(freshSummary, "(cache ") {
		t.Fatalf("expected fresh-cache summary not to contain stale cache caveat, got:\n%s", freshSummary)
	}

	// Stale cache (4h old): PR/forge-derived worktree labels carry "(cache 4h)" and Worktrees section prints the cache-age caveat,
	// while git-derived fields and git-only statuses ("default branch", "merged (ancestor of main)") have no cache caveat.
	st.BoardSyncedAt = time.Now().UTC().Add(-4 * time.Hour)
	staleSummary := BuildPickupSummary(st)
	for _, want := range []string{
		"Worktrees: 5\n  labels from forge cache synced 4h ago; run ateam state pickup",
		"- main (head=abad85b, dirty=false, ahead=0, default branch)",
		"- claude/ancestor (head=9a8b7c6, dirty=false, ahead=0, merged (ancestor of main))",
		"- gemini/fresh (head=abad85b, dirty=false, ahead=0, no commits yet)",
		"- codex/mercury-offline-contract (head=e50ca10f, dirty=false, ahead=4, open PR #64 (cache 4h))",
		"- codex/review-pr-79 (head=d7bb7968, dirty=false, ahead=5, no PR (cache 4h))",
	} {
		if !strings.Contains(staleSummary, want) {
			t.Fatalf("expected stale-cache pickup summary to contain %q, got:\n%s", want, staleSummary)
		}
	}

	// Mixed-provider case 1: Gitea active & fresh (2m ago), GitHub stale (4h ago).
	// GH-PR-64 (Forge: "github") must carry "(cache 4h)", GT-PR-12 (Forge: "gitea") and "no PR" (using active Gitea) must be uncaveated,
	// and the section caveat must report 4h.
	mixed := defaultState()
	_ = mixed.UpsertForge(ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"})
	_ = mixed.UpsertForge(ForgeConfig{Provider: ForgeGitea, BaseURL: "https://gitea.example.com", Owner: "c360", Repo: "ateam"})
	mixed.Forge = mixed.ResolveForge(ForgeGitea)
	mixed.ForgeSyncedAt = map[ForgeProvider]time.Time{
		ForgeGitHub: time.Now().UTC().Add(-4 * time.Hour),
		ForgeGitea:  time.Now().UTC().Add(-2 * time.Minute),
	}
	mixed.BoardSyncedAt = mixed.ForgeSyncedAt[ForgeGitea]
	mixed.Git.Branch = "main"
	mixed.Git.HeadSHA = "abad85b"
	mixed.Git.Worktrees = []Worktree{
		{Path: "/repo/.worktrees/wt-gh", Branch: "codex/gh-branch", Head: "e50ca10f", Dirty: &dirtyFalse, AheadCount: 4},
		{Path: "/repo/.worktrees/wt-gt", Branch: "codex/gt-branch", Head: "11223344", Dirty: &dirtyFalse, AheadCount: 2},
		{Path: "/repo/.worktrees/wt-nopr", Branch: "codex/review-pr-80", Head: "d7bb7968", Dirty: &dirtyFalse, AheadCount: 1},
	}
	mixed.Cards = []BoardCard{
		{ID: "GH-PR-64", Number: 64, Type: "pr", Column: "in_progress", Branch: "codex/gh-branch", HeadSHA: "e50ca10f", Forge: "github"},
		{ID: "GT-PR-12", Number: 12, Type: "pr", Column: "in_progress", Branch: "codex/gt-branch", HeadSHA: "11223344", Forge: "gitea"},
	}
	mixedSummary1 := BuildPickupSummary(mixed)
	for _, want := range []string{
		"Worktrees: 3\n  labels from forge cache synced 4h ago; run ateam state pickup",
		"- codex/gh-branch (head=e50ca10f, dirty=false, ahead=4, open PR #64 (cache 4h))",
		"- codex/gt-branch (head=11223344, dirty=false, ahead=2, open PR #12)",
		"- codex/review-pr-80 (head=d7bb7968, dirty=false, ahead=1, no PR)",
	} {
		if !strings.Contains(mixedSummary1, want) {
			t.Fatalf("expected mixed-provider summary (GitHub stale 4h, Gitea fresh 2m) to contain %q, got:\n%s", want, mixedSummary1)
		}
	}

	// Mixed-provider case 2 (reversed ages): Gitea active & stale (3h ago), GitHub fresh (2m ago).
	mixed.ForgeSyncedAt[ForgeGitHub] = time.Now().UTC().Add(-2 * time.Minute)
	mixed.ForgeSyncedAt[ForgeGitea] = time.Now().UTC().Add(-3 * time.Hour)
	mixed.BoardSyncedAt = mixed.ForgeSyncedAt[ForgeGitea]
	mixedSummary2 := BuildPickupSummary(mixed)
	for _, want := range []string{
		"Worktrees: 3\n  labels from forge cache synced 3h ago; run ateam state pickup",
		"- codex/gh-branch (head=e50ca10f, dirty=false, ahead=4, open PR #64)",
		"- codex/gt-branch (head=11223344, dirty=false, ahead=2, open PR #12 (cache 3h))",
		"- codex/review-pr-80 (head=d7bb7968, dirty=false, ahead=1, no PR (cache 3h))",
	} {
		if !strings.Contains(mixedSummary2, want) {
			t.Fatalf("expected mixed-provider summary (GitHub fresh 2m, Gitea stale 3h) to contain %q, got:\n%s", want, mixedSummary2)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestPopulateMCPProvenanceAndRegisterMCPProcess(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeAteam := filepath.Join(binDir, "ateam")
	if err := os.WriteFile(fakeAteam, []byte("#!/bin/sh\necho 'ateam v0.1.0 (e37563b)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cleanup := RegisterMCPProcess(dir, "7606b7e", "7606b7e111111111111111111111111111111111")
	live := LoadLiveMCPProcesses(dir)
	if len(live) != 1 || live[0].PID != os.Getpid() || live[0].BuildCommit != "7606b7e" {
		t.Fatalf("expected registered MCP process for current PID, got: %+v", live)
	}
	cleanup()
	if after := LoadLiveMCPProcesses(dir); len(after) != 0 {
		t.Fatalf("expected MCP process list to be empty after cleanup, got: %+v", after)
	}

	st := defaultState()
	st.Git.Branch = "main"
	st.Git.HeadSHA = "e37563b"
	st.PopulateMCPProvenance(context.Background(), dir, "7606b7e")

	if st.MCPBuild != "7606b7e" || st.DiskBuild != "e37563b" || st.MCPStaleTarget != "e37563b" {
		t.Fatalf("unexpected MCP provenance fields: MCPBuild=%q DiskBuild=%q MCPStaleTarget=%q", st.MCPBuild, st.DiskBuild, st.MCPStaleTarget)
	}
	summary := BuildPickupSummary(st)
	for _, want := range []string{
		"build=e37563b mcp=7606b7e",
		"WARNING: MCP server binary is stale; restart the client to pick up e37563b",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected pickup summary to contain %q, got:\n%s", want, summary)
		}
	}

	stFresh := defaultState()
	stFresh.Git.Branch = "main"
	stFresh.Git.HeadSHA = "e37563b"
	stFresh.PopulateMCPProvenance(context.Background(), dir, "e37563b")
	if stFresh.MCPStaleTarget != "" {
		t.Fatalf("expected matching MCP build not to set MCPStaleTarget, got %q", stFresh.MCPStaleTarget)
	}
	if strings.Contains(BuildPickupSummary(stFresh), "WARNING: MCP server binary is stale") {
		t.Fatalf("expected fresh MCP build not to warn, got:\n%s", BuildPickupSummary(stFresh))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestReview65ReusedBranchLatestMerge(t *testing.T) {
	dirtyFalse := false
	oldPR := BoardCard{ID: "GH-PR-1", Number: 1, Type: "pr", Column: "done", Merged: true, Branch: "gemini/reused-branch", HeadSHA: "aaaa1110000", Forge: "github"}
	newPR := BoardCard{ID: "GH-PR-2", Number: 2, Type: "pr", Column: "done", Merged: true, Branch: "gemini/reused-branch", HeadSHA: "bbbb2220000", Forge: "github"}

	for _, cards := range [][]BoardCard{
		{oldPR, newPR},
		{newPR, oldPR},
	} {
		st := defaultState()
		st.Git.Branch = "main"
		st.Git.HeadSHA = "e37563b"
		st.Git.Worktrees = []Worktree{
			{Path: "/repo/.worktrees/reused", Branch: "gemini/reused-branch", Head: "bbbb222", Dirty: &dirtyFalse, AheadCount: 2, MergedIntoMain: false},
		}
		st.Cards = append([]BoardCard(nil), cards...)
		st.ReconcileWorktrees()
		if !st.Git.Worktrees[0].MergedIntoMain {
			t.Fatalf("expected MergedIntoMain=true when PR #2 matches HEAD (order %v, %v), got false", cards[0].ID, cards[1].ID)
		}
		if got := st.WorktreePRStatus(st.Git.Worktrees[0]); got != "merged via #2" {
			t.Fatalf("WorktreePRStatus = %q, want %q", got, "merged via #2")
		}
	}

	// Verify cached-old -> synced-new MergeForgeCards history and post-merge commits ahead of #2:
	stHist := defaultState()
	stHist.Git.Branch = "main"
	stHist.Git.HeadSHA = "e37563b"
	stHist.Git.Worktrees = []Worktree{
		{Path: "/repo/.worktrees/reused", Branch: "gemini/reused-branch", Head: "bbbb222", Dirty: &dirtyFalse, AheadCount: 2, MergedIntoMain: false},
	}
	stHist.Cards = []BoardCard{oldPR}
	stHist.MergeForgeCards([]BoardCard{newPR, oldPR}, ForgeGitHub)
	if !stHist.Git.Worktrees[0].MergedIntoMain {
		t.Fatalf("expected MergeForgeCards to reconcile reused branch as MergedIntoMain=true when PR #2 matches HEAD")
	}
	if summary := BuildPickupSummary(stHist); !strings.Contains(summary, "- gemini/reused-branch (head=bbbb222, dirty=false, ahead=2, merged via #2)") {
		t.Fatalf("unexpected pickup summary after MergeForgeCards:\n%s", summary)
	}

	stHist.Git.Worktrees[0].Head = "cccc333"
	stHist.Git.Worktrees[0].AheadCount = 3
	stHist.ReconcileWorktrees()
	if stHist.Git.Worktrees[0].MergedIntoMain {
		t.Fatalf("expected MergedIntoMain=false when reused branch advances ahead of both merged PRs")
	}
	if got := stHist.WorktreePRStatus(stHist.Git.Worktrees[0]); got != "ahead of merged #2" {
		t.Fatalf("WorktreePRStatus = %q, want %q", got, "ahead of merged #2")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestReview65WorktreeRegistryVisibleFromMain(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	wtA := filepath.Join(base, "wt-a")
	wtB := filepath.Join(base, "wt-b")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	runGit(mainDir, "init", "-b", "main")
	runGit(mainDir, "config", "user.email", "test@example.com")
	runGit(mainDir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(mainDir, "add", "README.md")
	runGit(mainDir, "commit", "-m", "init")
	runGit(mainDir, "worktree", "add", "-b", "gemini/wt-a", wtA)
	runGit(mainDir, "worktree", "add", "-b", "codex/wt-b", wtB)

	cleanup := RegisterMCPProcess(wtA, "7606b7e", "7606b7e111111111111111111111111111111111")
	for name, root := range map[string]string{"main": mainDir, "wtA": wtA, "wtB": wtB} {
		live := LoadLiveMCPProcesses(root)
		if len(live) != 1 || live[0].PID != os.Getpid() || live[0].BuildCommit != "7606b7e" {
			t.Fatalf("LoadLiveMCPProcesses(%s) = %+v, want 1 record for current PID", name, live)
		}
	}
	cleanup()
	for name, root := range map[string]string{"main": mainDir, "wtA": wtA, "wtB": wtB} {
		if after := LoadLiveMCPProcesses(root); len(after) != 0 {
			t.Fatalf("LoadLiveMCPProcesses(%s) after cleanup = %+v, want empty", name, after)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestPerAgentBranchHandoffSlots(t *testing.T) {
	t.Setenv("ATEAM_AGENT", "")
	t.Setenv("ATEAM_SESSION_ID", "")
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("CODEX_SESSION_ID", "")
	t.Setenv("GEMINI_SESSION_ID", "")
	t.Setenv("CLAUDECODE", "")

	dir := t.TempDir()
	now := time.Now().UTC()

	// Precedence: explicit > ATEAM_AGENT > writer lease > branch prefix > "agent"
	if got := ResolveActiveAgent(dir, "codex", "claude/branch", &WriterLease{Agent: "gemini"}); got != "codex" {
		t.Fatalf("ResolveActiveAgent explicit = %q, want codex", got)
	}
	t.Setenv("ATEAM_AGENT", "gemini")
	if got := ResolveActiveAgent(dir, "", "claude/branch", &WriterLease{Agent: "codex"}); got != "gemini" {
		t.Fatalf("ResolveActiveAgent ATEAM_AGENT = %q, want gemini", got)
	}
	t.Setenv("ATEAM_AGENT", "")
	if got := ResolveActiveAgent(dir, "", "claude/branch", &WriterLease{Agent: "codex"}); got != "codex" {
		t.Fatalf("ResolveActiveAgent lease = %q, want codex", got)
	}
	if got := ResolveActiveAgent(dir, "", "claude/branch", nil); got != "claude" {
		t.Fatalf("ResolveActiveAgent branch prefix = %q, want claude", got)
	}
	if got := ResolveActiveAgent(dir, "", "main", nil); got != "agent" {
		t.Fatalf("ResolveActiveAgent default = %q, want agent", got)
	}

	st := defaultState()
	st.Git.Branch = "main"
	st.Git.HeadSHA = "9f11a42"
	st.Git.Worktrees = []Worktree{
		{Path: "/repo", Branch: "main", Head: "9f11a42"},
		{Path: "/repo/.worktrees/codex-71", Branch: "codex/mercury-usage-provenance", Head: "bb11cc2"},
		{Path: "/repo/.worktrees/gemini-70", Branch: "gemini/merged-slot", Head: "aa00bb1"},
	}
	st.Cards = []BoardCard{
		{ID: "GH-PR-71", Number: 71, Type: "pr", Column: "in-review", Branch: "codex/mercury-usage-provenance", Forge: "github"},
		{ID: "GH-PR-70", Number: 70, Type: "pr", Column: "done", Merged: true, Branch: "gemini/merged-slot", Forge: "github"},
	}

	// 1. Codex records a handoff on codex/mercury-usage-provenance (2h ago, PR #71)
	st.SetHandoff(Handoff{
		Goal:          "PR #71: finish mercury usage provenance",
		Addresses:     "PR #71",
		Constraints:   "none",
		DoneWhen:      "task verify green",
		ChecksToRun:   "task verify",
		Agent:         "codex",
		Branch:        "codex/mercury-usage-provenance",
		CheckpointSHA: "bb11cc2",
		UpdatedAt:     now.Add(-2 * time.Hour),
	})
	// 2. Gemini records a handoff on gemini/merged-slot whose PR #70 is now done/merged (stale)
	st.SetHandoff(Handoff{
		Goal:          "PR #70: land merged slot",
		Addresses:     "PR #70",
		Constraints:   "none",
		DoneWhen:      "merged",
		ChecksToRun:   "task verify",
		Agent:         "gemini",
		Branch:        "gemini/merged-slot",
		CheckpointSHA: "aa00bb1",
		UpdatedAt:     now.Add(-3 * time.Hour),
	})
	// 3. Claude records a handoff on main
	st.SetHandoff(Handoff{
		Goal:          "Coordinate epic #76 on main",
		Addresses:     "issue #76",
		Constraints:   "Wait for PR #71",
		DoneWhen:      "All epic issues merged",
		ChecksToRun:   "task verify",
		Agent:         "claude",
		Branch:        "main",
		CheckpointSHA: "9f11a42",
		UpdatedAt:     now.Add(-5 * time.Minute),
	})

	if err := Save(dir, st); err != nil {
		t.Fatalf("Save: %v", err)
	}
	rawJSON, err := os.ReadFile(CacheFilePath(dir))
	if err != nil {
		t.Fatalf("ReadFile state.json: %v", err)
	}
	if !strings.Contains(string(rawJSON), `"claude@main"`) || !strings.Contains(string(rawJSON), `"codex@codex/mercury-usage-provenance"`) {
		t.Fatalf("state.json missing expected handoffs map keys:\n%s", string(rawJSON))
	}

	// Pickup as claude@main: drops stale gemini@gemini/merged-slot, prints claude@main first, then other handoffs line for codex
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loaded.CurrentAgent = "claude"
	loaded.RefreshHandoffStaleness()
	if _, exists := loaded.Handoffs["gemini@gemini/merged-slot"]; exists {
		t.Fatalf("expected stale gemini@gemini/merged-slot to be dropped on pickup, got %+v", loaded.Handoffs)
	}
	if loaded.LastHandoff == nil || loaded.LastHandoff.Goal != "Coordinate epic #76 on main" {
		t.Fatalf("expected LastHandoff alias for claude@main, got %+v", loaded.LastHandoff)
	}
	claudePickup := BuildPickupSummary(loaded)
	wantClaudePrefix := "Last Handoff:\nGoal: Coordinate epic #76 on main\nAddresses: issue #76\nConstraints: Wait for PR #71\nDone when: All epic issues merged\nChecks to run: task verify\nother handoffs: codex@codex/mercury-usage-provenance (2h ago, PR #71)"
	if !strings.Contains(claudePickup, wantClaudePrefix) {
		t.Fatalf("claude@main pickup mismatch:\ngot:\n%s\nwant substring:\n%s", claudePickup, wantClaudePrefix)
	}

	// Pickup as gemini@main (no handoff for gemini@main): prints "Last Handoff: none for gemini@main" + both other live handoffs sorted
	loaded.CurrentAgent = "gemini"
	loaded.RefreshHandoffStaleness()
	if loaded.LastHandoff != nil {
		t.Fatalf("expected LastHandoff=nil when gemini@main has no handoff, got %+v", loaded.LastHandoff)
	}
	geminiPickup := BuildPickupSummary(loaded)
	wantGeminiSection := strings.Join([]string{
		"Last Handoff: none for gemini@main",
		"other handoffs: claude@main (5m ago)",
		"other handoffs: codex@codex/mercury-usage-provenance (2h ago, PR #71)",
	}, "\n")
	if !strings.Contains(geminiPickup, wantGeminiSection) {
		t.Fatalf("gemini@main pickup mismatch:\ngot:\n%s\nwant substring:\n%s", geminiPickup, wantGeminiSection)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func FuzzPerAgentHandoffSlotHistory(f *testing.F) {
	f.Add([]byte{0, 4, 8, 9, 22, 27})
	f.Add([]byte{0, 3, 7, 12, 18, 28})
	f.Add([]byte{1, 5, 10, 19, 2, 29})

	agents := []string{"claude", "codex", "gemini"}
	branches := []string{"main", "codex/feat-a", "gemini/feat-b"}

	f.Fuzz(func(t *testing.T, ops []byte) {
		t.Setenv("ATEAM_AGENT", "")
		t.Setenv("ATEAM_SESSION_ID", "")
		t.Setenv("CLAUDE_SESSION_ID", "")
		t.Setenv("CODEX_SESSION_ID", "")
		t.Setenv("GEMINI_SESSION_ID", "")
		t.Setenv("CLAUDECODE", "")

		st := defaultState()
		branchHeads := map[string]string{
			"main":          "1111111",
			"codex/feat-a":  "2222222",
			"gemini/feat-b": "3333333",
		}
		closedPRs := map[int]bool{}
		syncGitAndCards := func(curBranch string) {
			st.Git.Branch = curBranch
			st.Git.HeadSHA = branchHeads[curBranch]
			st.Git.Worktrees = []Worktree{
				{Path: "/repo", Branch: "main", Head: branchHeads["main"]},
				{Path: "/repo/wt-a", Branch: "codex/feat-a", Head: branchHeads["codex/feat-a"]},
				{Path: "/repo/wt-b", Branch: "gemini/feat-b", Head: branchHeads["gemini/feat-b"]},
			}
			st.Cards = nil
			for prNum := 1; prNum <= 3; prNum++ {
				col := "in-review"
				if closedPRs[prNum] {
					col = "done"
				}
				st.Cards = append(st.Cards, BoardCard{
					ID:     fmt.Sprintf("GH-PR-%d", prNum),
					Number: prNum,
					Type:   "pr",
					Column: col,
					Merged: closedPRs[prNum],
					Forge:  "github",
				})
			}
		}
		syncGitAndCards("main")
		st.Handoffs = make(map[string]*Handoff)

		type refEntry struct {
			agent  string
			branch string
			prNum  int
			h      Handoff
		}
		refKey := func(ag, br string) string {
			return strings.ToLower(strings.TrimSpace(ag)) + "@" + strings.TrimSpace(br)
		}
		refSlots := make(map[string]refEntry)
		pruneRef := func() {
			for k, e := range refSlots {
				if e.h.CheckpointSHA != branchHeads[e.branch] || closedPRs[e.prNum] {
					delete(refSlots, k)
				}
			}
		}

		dir := t.TempDir()
		now := time.Now().UTC().Add(-10 * time.Minute)
		for i, b := range ops {
			if i >= 32 {
				break
			}
			ag := agents[int(b)%len(agents)]
			br := branches[(int(b)/3)%len(branches)]
			action := (int(b) / 9) % 4
			switch action {
			case 0: // SetHandoff
				prNum := (i % 3) + 1
				syncGitAndCards(br)
				st.CurrentAgent = ag
				h := Handoff{
					Goal:          fmt.Sprintf("PR #%d: step %d for %s", prNum, i, ag),
					Addresses:     fmt.Sprintf("PR #%d", prNum),
					Constraints:   "none",
					DoneWhen:      "task verify",
					ChecksToRun:   "task verify",
					Agent:         ag,
					Branch:        br,
					CheckpointSHA: branchHeads[br],
					UpdatedAt:     now,
				}
				st.SetHandoff(h)
				refSlots[refKey(ag, br)] = refEntry{agent: ag, branch: br, prNum: prNum, h: h}
			case 1: // Advance branch HEAD
				branchHeads[br] = fmt.Sprintf("sha%04d", i+1)
				syncGitAndCards(st.Git.Branch)
			case 2: // Close PR
				prNum := (int(b) % 3) + 1
				closedPRs[prNum] = true
				syncGitAndCards(st.Git.Branch)
			case 3: // Pickup as ag@br (with Save/Load round-trip)
				syncGitAndCards(br)
				if err := Save(dir, st); err != nil {
					t.Fatalf("Save: %v", err)
				}
				loaded, err := Load(dir)
				if err != nil {
					t.Fatalf("Load: %v", err)
				}
				st = loaded
				syncGitAndCards(br)
				st.CurrentAgent = ag
				st.PruneStaleHandoffs()
				pruneRef()
			}
		}

		// Final pickup check across all (agent, branch) pairs
		for _, ag := range agents {
			for _, br := range branches {
				syncGitAndCards(br)
				st.CurrentAgent = ag
				st.PruneStaleHandoffs()
				pruneRef()
				curKey := refKey(ag, br)
				summary := BuildPickupSummary(st)

				if len(st.Handoffs) != len(refSlots) {
					t.Fatalf("len(st.Handoffs)=%d != len(refSlots)=%d (st=%+v, ref=%+v)", len(st.Handoffs), len(refSlots), st.Handoffs, refSlots)
				}
				if wantCur, ok := refSlots[curKey]; ok {
					if st.LastHandoff == nil || st.LastHandoff.Goal != wantCur.h.Goal {
						t.Fatalf("slot %s: LastHandoff=%+v, want Goal=%q", curKey, st.LastHandoff, wantCur.h.Goal)
					}
					if !strings.Contains(summary, "Last Handoff:\nGoal: "+wantCur.h.Goal) {
						t.Fatalf("slot %s: summary missing current handoff Goal %q:\n%s", curKey, wantCur.h.Goal, summary)
					}
				} else {
					if st.LastHandoff != nil {
						t.Fatalf("slot %s: expected LastHandoff=nil, got %+v", curKey, st.LastHandoff)
					}
					if !strings.Contains(summary, "Last Handoff: none for "+curKey) {
						t.Fatalf("slot %s: summary missing 'Last Handoff: none for %s':\n%s", curKey, curKey, summary)
					}
				}
				var otherKeys []string
				for k := range refSlots {
					if k != curKey {
						otherKeys = append(otherKeys, k)
					}
				}
				sort.Strings(otherKeys)
				for _, k := range otherKeys {
					e := refSlots[k]
					wantLine := fmt.Sprintf("other handoffs: %s (10m ago, PR #%d)", k, e.prNum)
					if !strings.Contains(summary, wantLine) {
						t.Fatalf("slot %s: summary missing %q:\n%s", curKey, wantLine, summary)
					}
				}
			}
		}
	})
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestReview79HandoffPRLabelAndRefreshPreservation(t *testing.T) {
	t.Setenv("ATEAM_AGENT", "")
	t.Setenv("ATEAM_SESSION_ID", "")
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("CODEX_SESSION_ID", "")
	t.Setenv("GEMINI_SESSION_ID", "")
	t.Setenv("CLAUDECODE", "")

	now := time.Now().UTC()
	st := defaultState()
	st.Git.Branch = "main"
	st.Git.HeadSHA = "9f11a42"
	st.Git.Worktrees = []Worktree{
		{Path: "/repo", Branch: "main", Head: "9f11a42"},
		{Path: "/repo/wt-codex", Branch: "codex/feat-open", Head: "bb11cc2"},
	}
	st.Cards = []BoardCard{
		{ID: "GH-PR-45", Number: 45, Type: "pr", Column: "done", Merged: true, Branch: "gemini/old", Forge: "github"},
		{ID: "GH-PR-71", Number: 71, Type: "pr", Column: "in-review", Branch: "codex/feat-open", Forge: "github"},
	}

	// Finding 4: Markdown #140 anchor, bare issue #76, and closed context PR #45 must NOT produce a false PR label in other handoffs.
	st.SetHandoff(Handoff{
		Goal:          "Post-merge cleanup after PR #45 landed on main",
		Addresses:     "openspec/specs/harness/spec.md#140, issue #76; context: fixed earlier by PR #45",
		DoneWhen:      "PR #45 follow-up verified",
		ChecksToRun:   "task verify",
		Agent:         "gemini",
		Branch:        "main",
		CheckpointSHA: "9f11a42",
		UpdatedAt:     now.Add(-5 * time.Minute),
	})
	// Open branch PR #71 on codex/feat-open should be preferred even if Goal mentions closed PR #45 mid-sentence.
	st.SetHandoff(Handoff{
		Goal:          "Follow up after PR #45 on codex branch",
		Addresses:     "openspec/specs/harness/spec.md#140",
		DoneWhen:      "task verify",
		ChecksToRun:   "task verify",
		Agent:         "codex",
		Branch:        "codex/feat-open",
		CheckpointSHA: "bb11cc2",
		UpdatedAt:     now.Add(-2 * time.Hour),
	})
	st.SetHandoff(Handoff{
		Goal:          "Active claude work on main",
		Addresses:     "issue #76",
		DoneWhen:      "done",
		ChecksToRun:   "task verify",
		Agent:         "claude",
		Branch:        "main",
		CheckpointSHA: "9f11a42",
		UpdatedAt:     now.Add(-1 * time.Minute),
	})

	st.CurrentAgent = "claude"
	summary := BuildPickupSummary(st)
	if !strings.HasSuffix(summary, "other handoffs: gemini@main (5m ago)") {
		t.Fatalf("expected gemini@main without false PR #140 or closed PR #45 label, got:\n%s", summary)
	}
	if !strings.Contains(summary, "other handoffs: codex@codex/feat-open (2h ago, PR #71)") {
		t.Fatalf("expected codex@codex/feat-open to prefer open branch PR #71, got:\n%s", summary)
	}

	// Finding 2: Agent-agnostic Refresh on main must not wipe LastHandoff to nil when handoffs["agent@main"] is absent,
	// and same-session merged-PR handoff (#53) must survive Refresh + ReconcilePickup.
	dir := t.TempDir()
	stDisk := defaultState()
	stDisk.Git.Branch = "main"
	stDisk.Git.HeadSHA = "uncommitted"
	stDisk.Cards = []BoardCard{
		{ID: "GH-PR-45", Number: 45, Type: "pr", Column: "done", Merged: true, Forge: "github"},
	}
	stDisk.SetHandoff(Handoff{
		Goal:          "PR #45: verify post-merge state on main",
		Addresses:     "main @ uncommitted",
		DoneWhen:      "verified",
		ChecksToRun:   "task verify",
		Agent:         "claude",
		SessionID:     "sess-active",
		Branch:        "main",
		CheckpointSHA: "uncommitted",
		UpdatedAt:     now,
	})
	if err := Save(dir, stDisk); err != nil {
		t.Fatalf("Save: %v", err)
	}

	refreshed, err := Refresh(context.Background(), dir, false)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshed.LastHandoff == nil || refreshed.LastHandoff.Goal != "PR #45: verify post-merge state on main" {
		t.Fatalf("expected agent-agnostic Refresh to preserve LastHandoff alias on disk, got %+v", refreshed.LastHandoff)
	}
	if refreshed.Handoffs["claude@main"] == nil {
		t.Fatalf("expected Refresh not to prematurely delete claude@main before session is resolved")
	}

	ReconcilePickup(dir, refreshed, "claude", "sess-active")
	if refreshed.Handoffs["claude@main"] == nil || refreshed.LastHandoff == nil {
		t.Fatalf("expected ReconcilePickup in same session sess-active to preserve merged-PR handoff, got Handoffs=%+v LastHandoff=%+v", refreshed.Handoffs, refreshed.LastHandoff)
	}
	loadedDisk, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after ReconcilePickup: %v", err)
	}
	if loadedDisk.LastHandoff == nil || loadedDisk.LastHandoff.Goal != "PR #45: verify post-merge state on main" {
		t.Fatalf("expected .ateam/state.json LastHandoff alias preserved on disk, got %+v", loadedDisk.LastHandoff)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestReview79CodexPickupLockLeaseAndLegacyMigration(t *testing.T) {
	t.Setenv("ATEAM_AGENT", "")
	t.Setenv("ATEAM_SESSION_ID", "")
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("CODEX_SESSION_ID", "")
	t.Setenv("GEMINI_SESSION_ID", "")
	t.Setenv("CLAUDECODE", "")
	now := time.Now().UTC()

	// Codex review Finding 1: ReconcilePickup with an older pre-lock snapshot must revalidate live git HEAD
	// under the lock and never delete a newer checkpoint written after HEAD advanced.
	gitDir := t.TempDir()
	headsDir := filepath.Join(gitDir, ".git", "refs", "heads")
	if err := os.MkdirAll(headsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(headsDir, "main"), []byte("1111111000000000000000000000000000000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	preLockSnap := defaultState()
	preLockSnap.Git.Branch = "main"
	preLockSnap.Git.HeadSHA = "1111111"
	if err := Save(gitDir, preLockSnap); err != nil {
		t.Fatal(err)
	}
	// Simulate another process advancing HEAD to 2222222 and saving a fresh checkpoint before ReconcilePickup acquires the lock.
	if err := os.WriteFile(filepath.Join(headsDir, "main"), []byte("2222222000000000000000000000000000000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(gitDir, func(cur *State) error {
		cur.SyncBranchHead("main", "2222222")
		cur.SetHandoff(Handoff{
			Goal:          "Concurrent newer checkpoint at 2222222",
			Addresses:     "main @ 2222222",
			DoneWhen:      "done",
			ChecksToRun:   "task verify",
			Agent:         "claude",
			Branch:        "main",
			CheckpointSHA: "2222222",
			UpdatedAt:     time.Now().UTC(),
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ReconcilePickup(gitDir, preLockSnap, "codex", "codex-sess")
	afterConcurrent, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if afterConcurrent.Handoffs["claude@main"] == nil || afterConcurrent.Handoffs["claude@main"].CheckpointSHA != "2222222" {
		t.Fatalf("expected concurrent newer checkpoint at 2222222 to survive ReconcilePickup, got %+v", afterConcurrent.Handoffs)
	}

	// Codex review Finding 2: Foreign live lease or exited lease must not select or overwrite another agent's slot.
	foreignLease := &WriterLease{Agent: "claude", SessionID: "claude-sess", PID: os.Getpid(), LastSeen: time.Now().UTC()}
	if got := ResolveActiveAgentForSession("", "", "codex-sess", "codex/feat", foreignLease); got != "codex" {
		t.Fatalf("expected foreign session lease to be ignored in favor of branch prefix codex, got %q", got)
	}
	exitedLease := &WriterLease{Agent: "claude", SessionID: "", PID: 99999999, LastSeen: time.Now().UTC()}
	if got := ResolveActiveAgentForSession("", "", "", "codex/feat", exitedLease); got != "codex" {
		t.Fatalf("expected exited lease PID to be ignored in favor of branch prefix codex, got %q", got)
	}

	// Codex review Finding 3: Legacy cache (last_handoff without handoffs map) must migrate without losing the legacy slot
	// or presenting another agent's legacy handoff as the current agent's goal.
	legacyDir := t.TempDir()
	legacyJSON := `{"mode":"full","git":{"branch":"main","head_sha":"9f11a42"},"last_handoff":{"goal":"Legacy Claude goal","addresses":"main","done_when":"done","checks_to_run":"task verify","agent":"claude","branch":"main","checkpoint_sha":"9f11a42"}}`
	if err := os.MkdirAll(filepath.Join(legacyDir, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CacheFilePath(legacyDir), []byte(legacyJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	legacySt, err := Load(legacyDir)
	if err != nil {
		t.Fatal(err)
	}
	ReconcilePickup(legacyDir, legacySt, "codex", "codex-sess")
	legacySummary := BuildPickupSummary(legacySt)
	if !strings.Contains(legacySummary, "Last Handoff: none for codex@main") || !strings.Contains(legacySummary, "other handoffs: claude@main") {
		t.Fatalf("expected legacy claude@main not to be presented as codex's goal, got:\n%s", legacySummary)
	}
	legacySt.SetHandoff(Handoff{
		Goal:          "New Codex goal on main",
		Addresses:     "main",
		DoneWhen:      "done",
		ChecksToRun:   "task verify",
		Agent:         "codex",
		Branch:        "main",
		CheckpointSHA: "9f11a42",
		UpdatedAt:     now,
	})
	if legacySt.Handoffs["claude@main"] == nil || legacySt.Handoffs["codex@main"] == nil {
		t.Fatalf("expected both migrated claude@main and new codex@main to survive SetHandoff, got %+v", legacySt.Handoffs)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestReview79LegacyMigrationWithoutAgentField(t *testing.T) {
	t.Setenv("ATEAM_AGENT", "")
	t.Setenv("ATEAM_SESSION_ID", "")
	t.Setenv("CLAUDE_SESSION_ID", "")
	t.Setenv("CODEX_SESSION_ID", "")
	t.Setenv("GEMINI_SESSION_ID", "")
	t.Setenv("CLAUDECODE", "")
	now := time.Now().UTC()

	// Legacy JSON without agent field on branch claude/work, followed by branch switch to codex/work + Refresh + pickup + handoff.
	noAgentLegacyDir := t.TempDir()
	noAgentJSON := `{"mode":"full","git":{"branch":"codex/work","head_sha":"bbbb222","worktrees":[{"path":"/tmp/wt-claude","branch":"claude/work","head":"aaaa111"}]},"last_handoff":{"goal":"Unattributed legacy goal","addresses":"claude/work","done_when":"done","checks_to_run":"task verify","branch":"claude/work","checkpoint_sha":"aaaa111"}}`
	if err := os.MkdirAll(filepath.Join(noAgentLegacyDir, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CacheFilePath(noAgentLegacyDir), []byte(noAgentJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	noAgentSt, err := Load(noAgentLegacyDir)
	if err != nil {
		t.Fatal(err)
	}
	noAgentSt.RefreshHandoffStaleness()
	ReconcilePickup(noAgentLegacyDir, noAgentSt, "codex", "codex-sess")
	noAgentSt.SetHandoff(Handoff{
		Goal:          "Codex goal on codex/work",
		Addresses:     "codex/work",
		DoneWhen:      "done",
		ChecksToRun:   "task verify",
		Agent:         "codex",
		Branch:        "codex/work",
		CheckpointSHA: "bbbb222",
		UpdatedAt:     now,
	})
	if err := Save(noAgentLegacyDir, noAgentSt); err != nil {
		t.Fatal(err)
	}
	reloadedNoAgent, err := Load(noAgentLegacyDir)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedNoAgent.Handoffs["claude@claude/work"] == nil || reloadedNoAgent.Handoffs["codex@codex/work"] == nil {
		t.Fatalf("expected legacy claude@claude/work without agent field and new codex@codex/work to survive branch switch + Refresh + SetHandoff, got %+v", reloadedNoAgent.Handoffs)
	}

	// Unprefixed branch ("feature/legacy") without agent field also survives branch switch under unknown-owner slot "agent@feature/legacy".
	unprefixedDir := t.TempDir()
	unprefixedJSON := `{"mode":"full","git":{"branch":"codex/work","head_sha":"bbbb222","worktrees":[{"path":"/tmp/wt-feat","branch":"feature/legacy","head":"aaaa111"}]},"last_handoff":{"goal":"Unprefixed legacy goal","addresses":"feature/legacy","done_when":"done","checks_to_run":"task verify","branch":"feature/legacy","checkpoint_sha":"aaaa111"}}`
	if err := os.MkdirAll(filepath.Join(unprefixedDir, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CacheFilePath(unprefixedDir), []byte(unprefixedJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	unprefixedSt, err := Load(unprefixedDir)
	if err != nil {
		t.Fatal(err)
	}
	unprefixedSt.RefreshHandoffStaleness()
	ReconcilePickup(unprefixedDir, unprefixedSt, "codex", "codex-sess")
	unprefixedSt.SetHandoff(Handoff{
		Goal:          "Codex goal on codex/work",
		Addresses:     "codex/work",
		DoneWhen:      "done",
		ChecksToRun:   "task verify",
		Agent:         "codex",
		Branch:        "codex/work",
		CheckpointSHA: "bbbb222",
		UpdatedAt:     now,
	})
	if err := Save(unprefixedDir, unprefixedSt); err != nil {
		t.Fatal(err)
	}
	reloadedUnprefixed, err := Load(unprefixedDir)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedUnprefixed.Handoffs["agent@feature/legacy"] == nil || reloadedUnprefixed.Handoffs["codex@codex/work"] == nil {
		t.Fatalf("expected legacy agent@feature/legacy and new codex@codex/work to survive branch switch, got %+v", reloadedUnprefixed.Handoffs)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestReadDiskBinaryBuildOutlivesTheOldTwoSecondBudget(t *testing.T) {
	// Regression for #116: a probe that answers after the old 2s budget but
	// well inside the 10s one must still be read, not fall back silently.
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The marker path reaches the script through the environment, never through
	// shell source: TMPDIR may contain spaces or apostrophes.
	started := filepath.Join(dir, "started")
	t.Setenv("ATEAM_TEST_STARTED", started)
	script := "#!/bin/sh\n: > \"$ATEAM_TEST_STARTED\"\nsleep 3\necho 'ateam v0.1.0 (e37563b)'\n"
	if err := os.WriteFile(filepath.Join(binDir, "ateam"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	shortRev, label := ReadDiskBinaryBuild(context.Background(), dir)
	if _, err := os.Stat(started); err != nil {
		t.Fatalf("probe never started: %v", err)
	}
	if shortRev != "e37563b" || label != "e37563b" {
		t.Fatalf("slow probe fell back: got (%q, %q), want (\"e37563b\", \"e37563b\")", shortRev, label)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestDiskBuildFailureIsNotReportedAsTheRunningBuild(t *testing.T) {
	// #121: a failed disk-binary read must say so, not copy the running MCP build
	// into DiskBuild and hide the stale-binary warning.
	for _, tc := range []struct {
		name, script, want string
	}{
		{"no binary", "", "absent"},
		{"probe exits non-zero", "#!/bin/sh\nexit 1\n", "unreadable"},
		{"version without parentheses", "#!/bin/sh\necho 'ateam v0.1.0'\n", "unreadable"},
		{"blank build in parentheses", "#!/bin/sh\necho 'ateam v0.1.0 (   )'\n", "unreadable"},
		{"empty build= in parentheses", "#!/bin/sh\necho 'ateam v0.1.0 (build=)'\n", "unreadable"},
		{"only -dirty in parentheses", "#!/bin/sh\necho 'ateam v0.1.0 (-dirty)'\n", "unreadable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.script != "" {
				if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "bin", "ateam"), []byte(tc.script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			st := defaultState()
			st.Git.Branch = "main"
			st.Git.HeadSHA = "e37563b"
			st.PopulateMCPProvenance(context.Background(), dir, "7606b7e")
			if st.DiskBuild != tc.want || st.MCPStaleTarget != "e37563b" {
				t.Fatalf("DiskBuild=%q MCPStaleTarget=%q, want DiskBuild=%q and the HEAD fallback e37563b", st.DiskBuild, st.MCPStaleTarget, tc.want)
			}
			if summary := BuildPickupSummary(st); !strings.Contains(summary, "build="+tc.want+" mcp=7606b7e") {
				t.Fatalf("pickup header does not show build=%s:\n%s", tc.want, summary)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestSetHandoffGuardedRefusesAnotherSessionsFreshHandoff(t *testing.T) {
	// #135: a deliberate handoff must not silently replace a fresh one from a
	// different or unknown session in the same agent@branch slot.
	held := Handoff{Goal: "session A goal", Agent: "claude", Branch: "main", CheckpointSHA: "1111111", UpdatedAt: time.Now().UTC()}
	for _, tc := range []struct {
		name         string
		holder       string // existing slot owner; "-" = empty slot
		holderGoal   string
		holderStale  bool
		writer       string
		force        bool
		wantRefusal  bool
		wantInSlotIs string
	}{
		{"different session", "A", "", false, "B", false, true, "session A goal"},
		{"unknown writer", "A", "", false, "", false, true, "session A goal"},
		{"unknown holder", "", "", false, "B", false, true, "session A goal"},
		{"different session forced", "A", "", false, "B", true, false, "B goal"},
		{"same session", "A", "", false, "A", false, false, "B goal"},
		{"empty slot", "-", "", false, "B", false, false, "B goal"},
		{"stale holder", "A", "", true, "B", false, false, "B goal"},
		{"auto-save holder", "A", AutoSaveGoal, false, "B", false, false, "B goal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := defaultState()
			st.Git.Branch = "main"
			st.Git.HeadSHA = "1111111"
			if tc.holder != "-" {
				h := held
				h.SessionID = tc.holder
				if tc.holderGoal != "" {
					h.Goal = tc.holderGoal
				}
				if tc.holderStale {
					// HEAD moved since the holder's checkpoint: RefreshHandoffStaleness marks it stale.
					h.CheckpointSHA = "2222222"
				}
				st.SetHandoff(h)
			}
			err := st.SetHandoffGuarded(Handoff{Goal: "B goal", Agent: "claude", SessionID: tc.writer, CheckpointSHA: "1111111", UpdatedAt: time.Now().UTC()}, tc.force)
			var refusal *HandoffSlotHeldError
			if gotRefusal := errors.As(err, &refusal); gotRefusal != tc.wantRefusal || (err != nil && !gotRefusal) {
				t.Fatalf("SetHandoffGuarded err = %v, want refusal=%v", err, tc.wantRefusal)
			}
			if got := st.Handoffs["claude@main"].Goal; got != tc.wantInSlotIs {
				t.Fatalf("slot goal = %q, want %q", got, tc.wantInSlotIs)
			}
			if tc.wantRefusal && (!strings.Contains(err.Error(), "handoff slot claude@main is held by") || !strings.Contains(err.Error(), `"session A goal"`)) {
				t.Fatalf("refusal does not name the slot, holder and goal: %v", err)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestPickupShowsHandoffsRecordedInsideWorktrees(t *testing.T) {
	// #136: each linked worktree has its own .ateam, so pickup from main must read them.
	mainDir, fresh, stale, broken := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	write := func(dir, branch, sha, goal string) {
		wt := defaultState()
		wt.Git.Branch, wt.Git.HeadSHA = branch, sha
		wt.SetHandoff(Handoff{Goal: goal, Agent: "claude", Branch: branch, CheckpointSHA: sha, UpdatedAt: time.Now().UTC()})
		if err := Save(dir, wt); err != nil {
			t.Fatal(err)
		}
	}
	// The current checkout's own file is skipped even when st no longer holds its (stale) slot.
	write(mainDir, "main", "0000000", "current checkout goal")
	write(fresh, "claude/fresh", "aaaaaaa", "fresh worktree goal")
	write(stale, "claude/stale", "bbbbbbb", "stale worktree goal")
	if err := os.MkdirAll(filepath.Join(broken, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CacheFilePath(broken), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := defaultState()
	st.Git.Branch, st.Git.HeadSHA = "main", "1111111"
	st.Git.Worktrees = []Worktree{
		{Path: mainDir, Branch: "main", Head: "1111111"},
		{Path: fresh, Branch: "claude/fresh", Head: "aaaaaaa"},
		{Path: stale, Branch: "claude/stale", Head: "ccccccc"}, // moved past its checkpoint
		{Path: broken, Branch: "claude/broken", Head: "ddddddd"},
	}
	summary := BuildPickupSummary(st)
	for _, want := range []string{
		"  - claude/fresh (head=aaaaaaa",
		"    handoff: fresh worktree goal (claude@claude/fresh, ",
		"    handoff: stale worktree goal (claude@claude/stale, ",
		", stale)",
		"Last Handoff: none for ",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("pickup summary missing %q:\n%s", want, summary)
		}
	}
	if strings.Contains(lineWithPrefixIn(summary, "    handoff: fresh worktree goal"), ", stale") || strings.Count(summary, "    handoff:") != 2 {
		t.Fatalf("expected exactly the two worktree handoffs, fresh not marked stale:\n%s", summary)
	}
}

func lineWithPrefixIn(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}
