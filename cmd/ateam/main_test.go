package main

import (
	"testing"
)

func TestCLISubcommands(t *testing.T) {
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("run(--help) = %d, want 0", code)
	}
	if code := run([]string{"version"}); code != 0 {
		t.Fatalf("run(version) = %d, want 0", code)
	}
	if code := run([]string{"doctor", "--check"}); code != 0 {
		t.Fatalf("run(doctor --check) = %d, want 0", code)
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
