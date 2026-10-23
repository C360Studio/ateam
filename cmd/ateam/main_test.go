package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/state"
)

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
	var giteaPullCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test/repo/issues":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test/repo/pulls":
			giteaPullCalls++
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
	if giteaPullCalls == 0 || !afterLive.SyncedAtFor(gtLive).After(old) {
		t.Fatalf("expected recovery %q to refresh Gitea (calls=%d, syncedAt=%v, old=%v)", command, giteaPullCalls, afterLive.SyncedAtFor(gtLive), old)
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
