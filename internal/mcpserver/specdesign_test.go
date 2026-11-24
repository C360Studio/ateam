package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Design-track trigger paths
func TestMCPSpecCheckAppliesDesignPaths(t *testing.T) {
	// #401 review: the MCP spec_check action must fail exactly what the CLI fails.
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".agents/review-policy.json", `{"version":1,"model_tiers":{"frontier":["claude:opus"]},
"effort_bands":{"claude":{"high":["high"]},"codex":{"high":["high"]},"gemini":{"high":["high"]}},
"cross_provider":false,"design_paths":["src/arming.rs"]}`)
	write("src/arming.rs", "fn arm() {}\n")
	write("openspec/config.yaml", "schema: spec-driven\n")
	git("init", "-q", "-b", "main")
	git("config", "user.email", "t@example.com")
	git("config", "user.name", "t")
	git("add", ".")
	git("commit", "-qm", "base")
	git("update-ref", "refs/remotes/origin/main", "HEAD")
	git("checkout", "-qb", "feature")
	write("src/arming.rs", "fn arm() { disarm() }\n")

	args := json.RawMessage(`{"action":"spec_check"}`)
	text, isErr := execStateTool(context.Background(), dir, args, "", "test")
	if !isErr || !strings.Contains(text, "src/arming.rs matches design_paths") {
		t.Fatalf("trigger path without design: isErr=%v text=%q, want an error naming the path", isErr, text)
	}
	write("openspec/changes/archive/2026-11-24-arm/design.md", "# Design\n")
	text, isErr = execStateTool(context.Background(), dir, args, "", "test")
	if isErr || !strings.Contains(text, "design paths: ok") {
		t.Fatalf("with design: isErr=%v text=%q, want the ok line", isErr, text)
	}
}
