package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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
}
