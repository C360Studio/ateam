package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/bootstrap"
)

// spec: openspec/changes/codex-hook-trust-311/specs/harness/spec.md#Requirement: Doctor reports Codex project hook trust
func TestDoctorReportsCodexHookTrust(t *testing.T) {
	dir, codexHome := t.TempDir(), t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("PATH", t.TempDir())
	if _, err := bootstrap.InitRepo(context.Background(), dir, bootstrap.Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte("model = \"o4\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out := captureStdout(t, func() int { return runDoctor(dir, []string{"--check"}) })
	var report struct {
		CodexHookTrust string   `json:"codex_hook_trust"`
		Warnings       []string `json:"warnings"`
		Violations     []string `json:"violations"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("doctor output: %v\n%s", err, out)
	}
	if report.CodexHookTrust != "untrusted" {
		t.Fatalf("codex_hook_trust = %q, want untrusted", report.CodexHookTrust)
	}
	has := func(list []string) bool {
		for _, s := range list {
			if strings.Contains(s, "Codex has not trusted") {
				return true
			}
		}
		return false
	}
	if !has(report.Warnings) || has(report.Violations) {
		t.Fatalf("want the trust finding as a warning only: warnings %q, violations %q", report.Warnings, report.Violations)
	}
}
