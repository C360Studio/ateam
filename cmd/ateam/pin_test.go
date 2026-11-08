package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/specs/harness/spec.md#Requirement: Doctor checks a declared ateam pin
func TestDoctorReportsDeclaredPin(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	t.Setenv("PATH", t.TempDir()) // no hook binary resolves, as on a CI runner
	doctor := func() (code int, report map[string]any) {
		t.Helper()
		code, out := captureStdout(t, func() int { return runDoctor(dir, []string{"--check"}) })
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("doctor output: %v\n%s", err, out)
		}
		return code, report
	}
	violations := func(report map[string]any) []string {
		var out []string
		for _, v := range report["violations"].([]any) {
			out = append(out, v.(string))
		}
		return out
	}

	_, report := doctor()
	if _, ok := report["pin"]; ok {
		t.Fatalf("doctor reports a pin without %s: %v", ".agents/ateam-version", report["pin"])
	}
	before := violations(report)

	if err := os.MkdirAll(filepath.Join(dir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".agents", "ateam-version"), []byte("v99.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, report := doctor()
	if report["pin"] != "v99.0.0" {
		t.Fatalf("pin = %v, want v99.0.0", report["pin"])
	}
	want := "running ateam is " + state.ReleaseVersion() + ", not the .agents/ateam-version pin v99.0.0"
	if got := violations(report); !slices.Contains(got, want) || len(got) != len(before)+1 {
		t.Fatalf("violations = %q, want %q added to %q", got, want, before)
	}
	if code == 0 || report["passed"] != false {
		t.Fatalf("doctor passed with a pin mismatch: code %d, passed %v", code, report["passed"])
	}
}
