package main

import (
	"context"
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
	if code := runStateHandoff(handoffDir, []string{"--goal", "CLI session A checkpoint", "--done-when", "verified", "--no-post"}); code != 0 {
		t.Fatalf("runStateHandoff = %d, want 0", code)
	}
	loaded, err := state.Load(handoffDir)
	if err != nil || loaded.LastHandoff == nil || loaded.LastHandoff.SessionID != "cli-session-A" {
		t.Fatalf("expected CLI handoff SessionID=cli-session-A, got %+v (err=%v)", loaded.LastHandoff, err)
	}
	t.Setenv("ATEAM_SESSION_ID", "")
	compactOut, err := hook.Handle(handoffDir, "pre-compact", strings.NewReader(`{"hook_event_name":"PreCompress","session_id":"cli-session-B"}`))
	if err != nil {
		t.Fatalf("hook.Handle pre-compact: %v", err)
	}
	if strings.Contains(compactOut.AdditionalContext, "CLI session A checkpoint") {
		t.Fatalf("session-B pre-compact preserved CLI session-A handoff:\n%s", compactOut.AdditionalContext)
	}
	if got := runStateSnapshot(handoffDir, "board"); got != 0 {
		t.Fatalf("runStateSnapshot(board) = %d, want 0", got)
	}
}
