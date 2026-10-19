package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestAgentSetupParity(t *testing.T) {
	violations := CheckParity(repoRoot(t))
	if len(violations) > 0 {
		t.Fatalf("repo agent setup parity violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

func TestInitRepoAndParitySensitivity(t *testing.T) {
	dir := t.TempDir()
	written, err := InitRepo(context.Background(), dir, Options{})
	if err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("InitRepo wrote 0 files")
	}
	if v := CheckParity(dir); len(v) > 0 {
		t.Fatalf("InitRepo output failed CheckParity:\n- %s", strings.Join(v, "\n- "))
	}

	// Sensitivity checks: prove that breaking each invariant is detected.
	t.Run("CLAUDE.md drops @AGENTS.md", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		_ = os.WriteFile(filepath.Join(sub, "CLAUDE.md"), []byte("# Custom rules\n"), 0o644)
		requireViolation(t, CheckParity(sub), "CLAUDE.md", "@AGENTS.md")
	})

	t.Run("AGENTS.md exceeds 24000 bytes", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		_ = os.WriteFile(filepath.Join(sub, "AGENTS.md"), []byte(strings.Repeat("a", maxAgentsMDBytes)), 0o644)
		requireViolation(t, CheckParity(sub), "AGENTS.md", "24000")
	})

	t.Run("AGENTS.md and rules-index.md row count mismatch", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		data, _ := os.ReadFile(filepath.Join(sub, "AGENTS.md"))
		edited := strings.Replace(string(data), "| `verify` | Go packages pass `gofmt`, `go vet`, and `revive` with zero warnings | `Taskfile.yml` |\n", "", 1)
		_ = os.WriteFile(filepath.Join(sub, "AGENTS.md"), []byte(edited), 0o644)
		requireViolation(t, CheckParity(sub), "AGENTS.md", ".agents/rules-index.md", "1. Ponytail decision ladder and code discipline")
	})

	t.Run("Claude skill adapter contains copied checkbox", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		p := filepath.Join(sub, ".claude", "skills", "pickup", "SKILL.md")
		data, _ := os.ReadFile(p)
		_ = os.WriteFile(p, append(data, []byte("- [ ] copied item\n")...), 0o644)
		requireViolation(t, CheckParity(sub), ".claude/skills/pickup/SKILL.md", "checkbox")
	})

	t.Run("Gemini hook config missing contextFileName", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		p := filepath.Join(sub, ".gemini", "settings.json")
		data, _ := os.ReadFile(p)
		edited := strings.Replace(string(data), `"contextFileName": "AGENTS.md",`, "", 1)
		_ = os.WriteFile(p, []byte(edited), 0o644)
		requireViolation(t, CheckParity(sub), ".gemini/settings.json", "contextFileName")
	})

	t.Run("Placeholder protocol or canonical skill rejected", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		_ = os.WriteFile(filepath.Join(sub, ".agents", "protocol.md"), []byte("# Shared Work Protocol\n\nClarify, Claim, Merge, Close.\n"), 0o644)
		_ = os.WriteFile(filepath.Join(sub, ".agents", "skills", "handoff", "SKILL.md"), []byte("---\nname: handoff\n---\n# handoff\nFollow protocol.\n"), 0o644)
		requireViolation(t, CheckParity(sub), ".agents/protocol.md", ".agents/skills/handoff/SKILL.md")
	})

	t.Run("README.md hook event drift rejected", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		_ = os.WriteFile(filepath.Join(sub, "README.md"), []byte("# Repo\n- `ateam hook <event>` (`session-start`, `pre-tool`, `post-tool`, `stop`)\n"), 0o644)
		requireViolation(t, CheckParity(sub), "README.md", "before-prompt")
	})
}

func requireViolation(t *testing.T, violations []string, substrings ...string) {
	t.Helper()
	joined := strings.Join(violations, "\n")
	for _, sub := range substrings {
		if !strings.Contains(joined, sub) {
			t.Fatalf("expected violation containing %q, got:\n%s", sub, joined)
		}
	}
}
