package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// spec: harness / Doctor checks the verify target
func TestDoctorFailsWithoutVerifyTarget(t *testing.T) {
	const noTarget = "Taskfile.yml: no verify target; the shipped skills and protocol run task verify"
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	doctor := func() (int, []string) {
		t.Helper()
		code, out := captureStdout(t, func() int { return runDoctor(dir, []string{"--check"}) })
		var report struct {
			Violations []string `json:"violations"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("doctor output: %v\n%s", err, out)
		}
		return code, report.Violations
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// The semsource shape from #415: doctor --check passed while task verify did not exist.
	write("version: '3'\ntasks:\n  check:\n    cmds:\n      - go test ./...\n")
	code, violations := doctor()
	if !slices.Contains(violations, noTarget) || code != 1 {
		t.Fatalf("doctor = %d, violations %q; want exit 1 and %q", code, violations, noTarget)
	}

	write("version: '3'\ntasks:\n  check:\n    cmds:\n      - go test ./...\n  verify:\n    cmds:\n      - task: check\n")
	if _, violations := doctor(); slices.Contains(violations, noTarget) {
		t.Fatalf("violations = %q, want no verify-target violation once verify exists", violations)
	}
}
