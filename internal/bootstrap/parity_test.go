package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestAgentSetupParity(t *testing.T) {
	violations := CheckParity(repoRoot(t))
	if len(violations) > 0 {
		t.Fatalf("repo agent setup parity violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
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

	t.Run("Antigravity hooks.json missing --agent agy", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		p := filepath.Join(sub, ".agents", "hooks.json")
		data, _ := os.ReadFile(p)
		edited := strings.ReplaceAll(string(data), "--agent agy", "--agent gemini")
		_ = os.WriteFile(p, []byte(edited), 0o644)
		requireViolation(t, CheckParity(sub), ".agents/hooks.json", "--agent agy")
	})

	t.Run("Antigravity plugin mcp_config.json missing mcp entry", func(t *testing.T) {
		sub := t.TempDir()
		_, _ = InitRepo(context.Background(), sub, Options{})
		p := filepath.Join(sub, ".agents", "plugins", "ateam", "mcp_config.json")
		_ = os.WriteFile(p, []byte(`{"mcpServers":{}}`), 0o644)
		requireViolation(t, CheckParity(sub), ".agents/plugins/ateam/mcp_config.json", "missing ateam mcp server entry")
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

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestDoctorBuildProvenanceAndDuplicateServeDetection(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	// 1. Build provenance drift check:
	if v := CheckBuildProvenance(ctx, dir, "ea0c7845a875", "9f11a42"); len(v) == 0 {
		t.Fatal("expected build provenance violation when build rev differs from HEAD")
	}
	if v := CheckBuildProvenance(ctx, dir, "9f11a4200000", "9f11a42"); len(v) > 0 {
		t.Fatalf("expected matching build provenance to pass, got: %v", v)
	}
	for _, emptyRev := range []string{"", "dev", "unknown"} {
		v := CheckBuildProvenance(ctx, dir, emptyRev, "9f11a42")
		if len(v) == 0 || !strings.Contains(v[0], "binary has no build provenance; build with task build") {
			t.Fatalf("expected missing build provenance warning for %q, got: %v", emptyRev, v)
		}
	}

	// 1b. Completed unarchived OpenSpec change warning:
	doneDir := filepath.Join(dir, "openspec", "changes", "done-change")
	_ = os.MkdirAll(doneDir, 0o755)
	_ = os.WriteFile(filepath.Join(doneDir, "tasks.md"), []byte("## 1. Tasks\n\n- [x] 1.1 Finished task\n"), 0o644)
	if w := CheckUnarchivedOpenSpecChanges(dir); len(w) != 1 || !strings.Contains(w[0], "openspec/changes/done-change is completed (1/1 tasks) and not archived") {
		t.Fatalf("expected unarchived OpenSpec change warning, got: %v", w)
	}

	// 2. Duplicate ateam serve process parsing and detection:
	psOut := strings.Join([]string{
		"  PID COMMAND",
		"54395 bin/ateam serve --addr 127.0.0.1:8421",
		"77267 ./bin/ateam serve --addr :8420",
		"88123 /usr/local/bin/ateam hook session-start",
	}, "\n")
	procs := ParseRunningServeProcesses(psOut, 99999)
	if len(procs) != 2 {
		t.Fatalf("expected 2 running serve processes, got %d: %+v", len(procs), procs)
	}
	if v := CheckDuplicateServeProcesses(procs); len(v) == 0 || !strings.Contains(v[0], "54395") || !strings.Contains(v[0], "77267") {
		t.Fatalf("expected duplicate serve process violation listing both PIDs, got: %v", v)
	}
	if v := CheckDuplicateServeProcesses(procs[:1]); len(v) > 0 {
		t.Fatalf("expected single serve process not to report violation, got: %v", v)
	}

	// 3. Running ateam mcp process parsing and stale build warning:
	mcpPS := strings.Join([]string{
		"  PID COMMAND",
		"41001 ./bin/ateam mcp",
		"41002 /repo/bin/ateam mcp",
	}, "\n")
	mcpProcs := ParseRunningMCPProcesses(mcpPS, 99999, []state.MCPProcess{
		{PID: 41001, BuildCommit: "7606b7e"},
	}, "e37563b")
	if len(mcpProcs) != 2 || mcpProcs[0].BuildCommit != "7606b7e" || mcpProcs[1].BuildCommit != "e37563b" {
		t.Fatalf("unexpected parsed MCP processes: %+v", mcpProcs)
	}
	mcpWarnings := CheckRunningMCPProcesses(mcpProcs, "e37563b")
	if len(mcpWarnings) != 1 || !strings.Contains(mcpWarnings[0], "PID 41001, build 7606b7e") {
		t.Fatalf("expected stale MCP warning only for PID 41001, got: %v", mcpWarnings)
	}
}
