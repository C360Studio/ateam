package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestBenchRejectsForgedRescoreAndCLIArguments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forged.json")
	if err := os.WriteFile(path, []byte(`{"selftest_passed":true,"tasks":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if runBench(dir, []string{"--rescore", path}) == 0 {
		t.Fatal("forged report accepted")
	}
	for _, args := range [][]string{{"run"}, {"compare", path}, {"rescore", path}, {"run", path, "--bogus"}} {
		if runExperimentCLI(context.Background(), dir, args) == 0 {
			t.Fatalf("accepted %v", args)
		}
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestBenchExportRejectsArgumentsBeforeReadingInput(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing-run.json")
	out := filepath.Join(dir, "out")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"missing artifact", []string{"export"}, "usage:"},
		{"missing destination", []string{"export", missing}, "exactly one"},
		{"conflicting destinations", []string{"export", missing, "--output", out, "--check", out}, "exactly one"},
		{"empty output", []string{"export", missing, "--output="}, "exactly one"},
		{"empty conflicting output", []string{"export", missing, "--output=", "--check", out}, "exactly one"},
		{"extra positional", []string{"export", missing, "--output", out, "extra"}, "unexpected export arguments"},
		{"unknown flag", []string{"export", missing, "--output", out, "--overwrite"}, "flag provided but not defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, diagnostic := captureBenchError(t, func() int {
				return runExperimentCLI(context.Background(), dir, tc.args)
			})
			if code == 0 || !strings.Contains(diagnostic, tc.want) {
				t.Fatalf("exit=%d diagnostic=%q; want failure containing %q", code, diagnostic, tc.want)
			}
		})
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Offline validated report projection
// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Explicit replay association and outcome
// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestBenchExportAndCheckCLI(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join("..", "..", "internal", "bench", "testdata", "report", "complete-run.json")
	output := filepath.Join(dir, "report")
	if code := runBench(dir, []string{"export", fixture, "--output", output}); code != 0 {
		t.Fatalf("export exit=%d", code)
	}
	for _, name := range []string{"results.json", "results.md", "evidence.json"} {
		if _, err := os.Stat(filepath.Join(output, name)); err != nil {
			t.Fatal(err)
		}
	}
	if code := runBench(dir, []string{"export", fixture, "--check", output}); code != 0 {
		t.Fatalf("check exit=%d", code)
	}
	if code := runBench(dir, []string{"export", fixture, "--output", output}); code == 0 {
		t.Fatal("existing output accepted")
	}
	if err := os.WriteFile(filepath.Join(output, "results.md"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := runBench(dir, []string{"export", fixture, "--check", output}); code == 0 {
		t.Fatal("changed report accepted")
	}
	retained, err := os.ReadFile(filepath.Join(output, "results.md"))
	if err != nil || string(retained) != "changed\n" {
		t.Fatalf("check changed report: %q, %v", retained, err)
	}
	if code := runBench(dir, []string{"export", fixture, "--replay", filepath.Join(dir, "missing-replay.json"), "--output", filepath.Join(dir, "replay")}); code == 0 {
		t.Fatal("missing replay ignored")
	}
	if _, err := os.Stat(filepath.Join(dir, "replay")); !os.IsNotExist(err) {
		t.Fatalf("invalid replay created output: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledOutput := filepath.Join(dir, "canceled")
	if code := runExperimentCLI(ctx, dir, []string{"export", fixture, "--output", canceledOutput}); code == 0 {
		t.Fatal("canceled export accepted")
	}
	if _, err := os.Stat(canceledOutput); !os.IsNotExist(err) {
		t.Fatalf("canceled export created output: %v", err)
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Offline validated report projection
func TestBenchExportRejectsInvalidInput(t *testing.T) {
	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{invalid, filepath.Join(dir, "missing.json")} {
		output := filepath.Join(dir, "report")
		if code := runBench(dir, []string{"export", input, "--output", output}); code == 0 {
			t.Fatalf("invalid input %q accepted", input)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("invalid input created output: %v", err)
		}
	}
}

func captureBenchError(t *testing.T, run func() int) (int, string) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	os.Stderr = file
	defer func() { os.Stderr = previous }()
	code := run()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return code, string(data)
}
