package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/test-attribution/spec.md#Requirement: Deterministic discovery and verification
func TestTestSpecCLI(t *testing.T) {
	root := t.TempDir()
	if code := runTestSpec(root, nil); code != 0 {
		t.Fatalf("empty tree: %d", code)
	}
	if err := os.WriteFile(filepath.Join(root, "bad_test.go"), []byte("package p\nfunc TestMissing(t *testing.T) {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if code := runTestSpec(root, nil); code != 1 {
		t.Fatalf("unattributed test: %d", code)
	}
	if code := runTestSpec(root, []string{"extra"}); code != 2 {
		t.Fatalf("invalid invocation: %d", code)
	}
	if code := runTestSpec(filepath.Join(root, "absent"), nil); code != 2 {
		t.Fatalf("scan failure: %d", code)
	}
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: Deterministic discovery and verification
func TestVerificationRunsAttributionGate(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := findRepoRoot(cwd)
	taskfile, err := filepath.Abs(filepath.Join(root, "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	dry := exec.Command("task", "--dry", "--taskfile", taskfile, "verify")
	out, err := dry.CombinedOutput()
	if err != nil {
		t.Fatalf("dry verify: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "go run ./cmd/ateam test-spec") || !strings.Contains(string(out), "npm --prefix ui run test:unit -- scripts/testspec.test.js") {
		t.Fatalf("attribution gate disconnected: %s", out)
	}
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	for _, name := range []string{"go", "gofmt", "revive", "node", "npm"} {
		script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$TESTSPEC_CALL_LOG\"\ncase \"$*\" in *test-spec*) exit 23;; esac\nexit 0\n"
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("task", "--taskfile", taskfile, "verify")
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "TESTSPEC_CALL_LOG="+log)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("verify swallowed attribution failure: %s", out)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "run ./cmd/ateam test-spec") || strings.Contains(string(data), "mod tidy") || strings.Contains(string(data), "--prefix") {
		t.Fatalf("verify did not stop at attribution: %s; %s", data, out)
	}
}
