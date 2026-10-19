package main

import (
	"context"
	"os"
	"path/filepath"
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
