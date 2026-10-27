package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/state"
)

// TestMain clears ATEAM_HOOK_SCOPE so a developer's exported knob does not change the doctor runs here, which
// inherit this environment, or let them read the developer's home directory.
func TestMain(m *testing.M) {
	if err := os.Unsetenv("ATEAM_HOOK_SCOPE"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestCLISubcommands(t *testing.T) {
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("run(--help) = %d, want 0", code)
	}
	if code := run([]string{"version"}); code != 0 {
		t.Fatalf("run(version) = %d, want 0", code)
	}
	// Doctor measures hook wall time internally. Measuring the race-instrumented
	// test binary adds instrumentation cost to that production latency gate.
	// Build/startup time is outside doctor's measurement; parity and its original
	// threshold still run against this repository, without relaxing the gate.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "ateam-doctor")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-race=false", "-o", binary, ".")
	build.WaitDelay = time.Second
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build native doctor: %v\n%s", err, output)
	}
	cmd := exec.CommandContext(ctx, binary, "doctor", "--check")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native doctor --check: %v\n%s", err, output)
	}
	if code := run([]string{"bench", "--selftest"}); code != 0 {
		t.Fatalf("run(bench --selftest) = %d, want 0", code)
	}
	if code := run([]string{"state", "spec-check"}); code != 0 {
		t.Fatalf("run(state spec-check) = %d, want 0", code)
	}
	if code := run([]string{"state", "spec-queue"}); code != 0 {
		t.Fatalf("run(state spec-queue) = %d, want 0", code)
	}
	if code := run([]string{"unknown-subcommand"}); code == 0 {
		t.Fatalf("expected non-zero exit code for unknown subcommand")
	}

	// Review Finding #3: CLI runStateHandoff populates SessionID from ATEAM_SESSION_ID,
	// and a subsequent session-B pre-compact replaces rather than claims session-A's handoff.
	handoffDir := t.TempDir()
	t.Setenv("ATEAM_SESSION_ID", "cli-session-A")
	if code := runStateHandoff(handoffDir, []string{"--agent", "claude", "--goal", "CLI session A checkpoint", "--done-when", "verified", "--no-post"}); code != 0 {
		t.Fatalf("runStateHandoff = %d, want 0", code)
	}
	loaded, err := state.Load(handoffDir)
	if err != nil || loaded.LastHandoff == nil || loaded.LastHandoff.SessionID != "cli-session-A" || loaded.LastHandoff.Agent != "claude" {
		t.Fatalf("expected CLI handoff SessionID=cli-session-A Agent=claude, got %+v (err=%v)", loaded.LastHandoff, err)
	}
	if loaded.Handoffs["claude@main"] == nil {
		t.Fatalf("expected Handoffs[claude@main] to be populated, got %+v", loaded.Handoffs)
	}
	t.Setenv("ATEAM_SESSION_ID", "")
	compactOut, err := hook.HandleWithAgent(handoffDir, "pre-compact", "claude", strings.NewReader(`{"hook_event_name":"PreCompress","session_id":"cli-session-B"}`))
	if err != nil {
		t.Fatalf("hook.Handle pre-compact: %v", err)
	}
	if strings.Contains(compactOut.AdditionalContext, "CLI session A checkpoint") {
		t.Fatalf("session-B pre-compact preserved CLI session-A handoff:\n%s", compactOut.AdditionalContext)
	}
	if got := runStatePickup(handoffDir, []string{"--agent", "claude"}); got != 0 {
		t.Fatalf("runStatePickup(--agent claude) = %d, want 0", got)
	}
	if got := runStateSnapshot(handoffDir, "board"); got != 0 {
		t.Fatalf("runStateSnapshot(board) = %d, want 0", got)
	}

	// Verify the recovery command advertised by stale forge-cache worktree output parses and succeeds through runState.
	dirtyFalse := false
	staleSt := &state.State{
		Mode:          state.ModeFull,
		BoardSyncedAt: time.Now().UTC().Add(-2 * time.Hour),
		Git: state.GitStatus{
			Branch:  "main",
			HeadSHA: "abad85b",
			Worktrees: []state.Worktree{
				{Path: handoffDir, Branch: "codex/feat", Head: "abad85b", Dirty: &dirtyFalse, AheadCount: 1},
			},
		},
	}
	staleSummary := state.BuildPickupSummary(staleSt)
	_, afterRun, ok := strings.Cut(staleSummary, "; run ateam ")
	if !ok {
		t.Fatalf("expected '; run ateam <cmd>' in stale worktree summary, got:\n%s", staleSummary)
	}
	cmdLine, _, _ := strings.Cut(afterRun, "\n")
	cmdFields := strings.Fields(cmdLine)
	if len(cmdFields) < 2 || cmdFields[0] != "state" {
		t.Fatalf("unexpected advertised command %q", cmdLine)
	}
	if got := runState(handoffDir, cmdFields[1:]); got != 0 {
		t.Fatalf("advertised recovery command %q exited with %d, want 0", cmdLine, got)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestRecoveryCommandRefreshesSupplyingForge(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-4 * time.Hour)
	wt := state.Worktree{Branch: "codex/gt-feature", Head: "1234567", AheadCount: 1}
	gh := state.ForgeConfig{Provider: state.ForgeGitHub, Owner: "test", Repo: "repo"}

	// Case 1: Inactive Gitea provider is reachable via httptest.Server -> recovery refreshes Gitea and clears (cache 4h).
	var giteaPullCalls atomic.Int32 // the sync fetches open and closed pulls concurrently
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test/repo/issues":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test/repo/pulls":
			giteaPullCalls.Add(1)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number":   12,
					"title":    "Gitea feature PR",
					"state":    "open",
					"draft":    false,
					"html_url": "http://gitea.local/test/repo/pulls/12",
					"head":     map[string]any{"ref": wt.Branch, "sha": wt.Head},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	gtLive := state.ForgeConfig{Provider: state.ForgeGitea, BaseURL: srv.URL, Owner: "test", Repo: "repo"}
	liveRoot := t.TempDir()
	stLive := &state.State{
		Mode:             state.ModeFull,
		Forge:            gh,
		Forges:           []state.ForgeConfig{gh, gtLive},
		ForgesConfigured: true,
		ForgeSyncedAt:    map[state.ForgeProvider]time.Time{state.ForgeGitHub: now, state.ForgeGitea: old},
		BoardSyncedAt:    now,
		Git:              state.GitStatus{Branch: "main", Worktrees: []state.Worktree{wt}},
		Cards:            []state.BoardCard{{ID: "GT-PR-12", Number: 12, Type: "pr", Forge: "gitea", Column: "in-progress", Branch: wt.Branch}},
	}
	summary := state.BuildPickupSummary(stLive)
	if !strings.Contains(summary, "open PR #12 (cache 4h)") {
		t.Fatalf("expected stale Gitea label before recovery, got:\n%s", summary)
	}
	_, recovery, ok := strings.Cut(summary, "; run ateam ")
	if !ok {
		t.Fatalf("expected recovery instruction in summary, got:\n%s", summary)
	}
	command, _, _ := strings.Cut(recovery, "\n")
	if err := state.Save(liveRoot, stLive); err != nil {
		t.Fatalf("Save(liveRoot): %v", err)
	}
	fields := strings.Fields(command)
	if code := runState(liveRoot, fields[1:]); code != 0 {
		t.Fatalf("recovery command %q failed with %d", command, code)
	}
	afterLive, err := state.Load(liveRoot)
	if err != nil {
		t.Fatalf("Load(liveRoot): %v", err)
	}
	if giteaPullCalls.Load() == 0 || !afterLive.SyncedAtFor(gtLive).After(old) {
		t.Fatalf("expected recovery %q to refresh Gitea (calls=%d, syncedAt=%v, old=%v)", command, giteaPullCalls.Load(), afterLive.SyncedAtFor(gtLive), old)
	}
	if gotStatus := afterLive.WorktreePRStatus(wt); gotStatus != "open PR #12" {
		t.Fatalf("expected fresh Gitea label %q after recovery, got %q", "open PR #12", gotStatus)
	}

	// Case 2: Inactive Gitea provider is unreachable -> recovery attempts Gitea and persists BoardSyncError.
	failRoot := t.TempDir()
	gtFail := state.ForgeConfig{Provider: state.ForgeGitea, BaseURL: "http://127.0.0.1:1", Owner: "test", Repo: "repo"}
	stFail := &state.State{
		Mode:             state.ModeFull,
		Forge:            gh,
		Forges:           []state.ForgeConfig{gh, gtFail},
		ForgesConfigured: true,
		ForgeSyncedAt:    map[state.ForgeProvider]time.Time{state.ForgeGitHub: now, state.ForgeGitea: old},
		BoardSyncedAt:    now,
		Git:              state.GitStatus{Branch: "main", Worktrees: []state.Worktree{wt}},
		Cards:            []state.BoardCard{{ID: "GT-PR-12", Number: 12, Type: "pr", Forge: "gitea", Column: "in-progress", Branch: wt.Branch}},
	}
	if err := state.Save(failRoot, stFail); err != nil {
		t.Fatalf("Save(failRoot): %v", err)
	}
	if code := runState(failRoot, fields[1:]); code != 0 {
		t.Fatalf("recovery command %q on unreachable forge exited %d, want 0", command, code)
	}
	afterFail, err := state.Load(failRoot)
	if err != nil {
		t.Fatalf("Load(failRoot): %v", err)
	}
	if afterFail.SyncedAtFor(gtFail).Equal(old) && afterFail.BoardSyncError == "" {
		t.Fatalf("advertised recovery %q succeeded without attempting the stale provider; label remains %q", command, afterFail.WorktreePRStatus(wt))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestStateHandoffRefusesAnotherSessionsFreshHandoff(t *testing.T) {
	// #135: the CLI refuses with exit 2 and leaves the slot alone; --force replaces it.
	dir := t.TempDir()
	t.Setenv("ATEAM_SESSION_ID", "cli-A")
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal A", "--no-post"}); code != 0 {
		t.Fatalf("session A handoff = %d, want 0", code)
	}
	t.Setenv("ATEAM_SESSION_ID", "cli-B")
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal B", "--no-post"}); code != 2 {
		t.Fatalf("session B handoff over A = %d, want 2", code)
	}
	if st, _ := state.Load(dir); st.Handoffs["claude@main"] == nil || st.Handoffs["claude@main"].Goal != "goal A" {
		t.Fatalf("refused handoff changed the slot: %+v", st.Handoffs["claude@main"])
	}
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal B", "--no-post", "--force"}); code != 0 {
		t.Fatalf("session B handoff --force = %d, want 0", code)
	}
	if st, _ := state.Load(dir); st.Handoffs["claude@main"].Goal != "goal B" {
		t.Fatalf("--force did not replace the slot: %+v", st.Handoffs["claude@main"])
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestInitFailureReportsBackups(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A regular file where a skill directory belongs fails init after the backup.
	if err := os.WriteFile(filepath.Join(root, ".agents", "skills"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stderr := captureBenchError(t, func() int { return runInit(root, []string{"-dir", root, "-force"}) })
	if code == 0 {
		t.Fatal("runInit succeeded although .agents/skills is a file")
	}
	backups, err := filepath.Glob(filepath.Join(root, "AGENTS.md.ateam-backup-*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups on disk = %v (%v), want one", backups, err)
	}
	want := "ateam init: backed up before failing: " + filepath.Base(backups[0])
	if !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Lifecycle hook normalization under 10ms
func TestHookLatencyFindings(t *testing.T) {
	tests := []struct {
		name          string
		us            int64
		wantWarning   bool
		wantViolation bool
	}{
		{"within target", 10000, false, false},
		{"just over target", 10001, true, false},
		{"old 5ms target no longer warns", 6000, false, false},
		{"at ceiling", 25000, true, false},
		{"over ceiling", 25001, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			warning, violation := hookLatencyFindings(tc.us)
			if (warning != "") != tc.wantWarning || (violation != "") != tc.wantViolation {
				t.Fatalf("hookLatencyFindings(%d) = (%q, %q), want warning=%v violation=%v", tc.us, warning, violation, tc.wantWarning, tc.wantViolation)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Default-branch handoff freshness follows the remote tip
func TestStateHandoffBehindOriginMainDoesNotHoldSlot(t *testing.T) {
	// #210: session A hands off from an unpulled main; origin/main then moves. Session B's handoff goes through.
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "A")
	a := git("rev-parse", "HEAD")
	git("update-ref", "refs/remotes/origin/main", a)
	t.Setenv("ATEAM_SESSION_ID", "cli-A")
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal A", "--no-post"}); code != 0 {
		t.Fatalf("session A handoff = %d, want 0", code)
	}
	t.Setenv("ATEAM_SESSION_ID", "cli-B")
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal B", "--no-post"}); code != 2 {
		t.Fatalf("session B handoff while A is at origin/main = %d, want 2", code)
	}
	// A commit lands on origin/main (a PR merged on the forge); the local checkout is not pulled.
	b := git("commit-tree", "-p", a, "-m", "B", git("rev-parse", "HEAD^{tree}"))
	git("update-ref", "refs/remotes/origin/main", b)
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal B", "--no-post"}); code != 0 {
		t.Fatalf("session B handoff after origin/main moved = %d, want 0", code)
	}
	if st, _ := state.Load(dir); st.Handoffs["claude@main"] == nil || st.Handoffs["claude@main"].Goal != "goal B" {
		t.Fatalf("slot after origin/main moved = %+v, want goal B", st.Handoffs["claude@main"])
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Default-branch handoff freshness follows the remote tip
func TestStateHandoffWarnsWithoutSessionID(t *testing.T) {
	for _, k := range []string{"ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CODEX_SESSION_ID", "GEMINI_SESSION_ID"} {
		t.Setenv(k, "")
	}
	const warning = "warning: handoff written without a session id; it will hold slot claude@main against every session"
	dir := t.TempDir()
	code, stderr := captureBenchError(t, func() int {
		return runStateHandoff(dir, []string{"--agent", "claude", "--goal", "goal", "--no-post"})
	})
	if code != 0 || !strings.Contains(stderr, warning) {
		t.Fatalf("id-less handoff = %d, stderr %q; want 0 and %q", code, stderr, warning)
	}
	code, stderr = captureBenchError(t, func() int {
		return runStateHandoff(dir, []string{"--agent", "claude", "--session", "S", "--goal", "goal", "--no-post", "--force"})
	})
	if code != 0 || strings.Contains(stderr, "without a session id") {
		t.Fatalf("handoff with a session id = %d, stderr %q; want 0 and no warning", code, stderr)
	}
}
