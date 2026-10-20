package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestReview65ReusedPIDIsNotMCP(t *testing.T) {
	// 1. Recorded PID 41001 was reused by a non-MCP command (/usr/bin/sleep 600):
	reusedNonMCP := strings.Join([]string{
		"  PID COMMAND",
		"41001 /usr/bin/sleep 600",
	}, "\n")
	if got := ParseRunningMCPProcesses(reusedNonMCP, 99999, []state.MCPProcess{
		{PID: 41001, BuildCommit: "abcdef0"},
	}, "unknown"); len(got) != 0 {
		t.Fatalf("expected reused non-MCP PID to be ignored, got: %+v", got)
	}

	// 2. Recorded PID 41001 was reused by a newer ateam mcp process whose lstart differs from StartedAt:
	oldStart := time.Date(2026, time.October, 20, 10, 0, 0, 0, time.Local).UTC()
	reusedMCP := "41001 Tue Oct 20 11:30:00 2026 ./bin/ateam mcp\n"
	got := ParseRunningMCPProcesses(reusedMCP, 99999, []state.MCPProcess{
		{PID: 41001, BuildCommit: "abcdef0", StartedAt: &oldStart},
	}, "unknown")
	if len(got) != 1 || got[0].BuildCommit != "unknown" {
		t.Fatalf("expected reused MCP PID with mismatched start time to fall back to unknown build, got: %+v", got)
	}

	// 3. Explicit unavailable warning when ps fails:
	emptyPathDir := t.TempDir()
	t.Setenv("PATH", emptyPathDir)
	procs, warnings := InspectRunningMCPProcesses(context.Background(), emptyPathDir, "b25a43f")
	if len(procs) != 0 || len(warnings) != 1 || !strings.Contains(warnings[0], "unable to inspect running ateam mcp processes via ps") {
		t.Fatalf("expected explicit ps unavailable warning, got procs=%+v warnings=%v", procs, warnings)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Lifecycle hook normalization under 5ms
// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestFilterAndCheckClientMCPProcesses(t *testing.T) {
	psTree := strings.Join([]string{
		"  PID  PPID STARTED                      COMMAND",
		"37111 79054 Tue Oct 20 12:00:00 2026 /Users/dev/.local/bin/claude",
		"37128 37111 Tue Oct 20 12:00:01 2026 ateam mcp",
		"37200 37111 Tue Oct 20 12:05:00 2026 /bin/sh -c ateam hook session-start",
		"37201 37200 Tue Oct 20 12:05:00 2026 ateam hook session-start",
		"72264 79054 Tue Oct 20 12:10:00 2026 /Users/dev/.local/bin/codex",
		"72369 72264 Tue Oct 20 12:10:01 2026 ./bin/ateam mcp",
	}, "\n")

	recorded := []state.MCPProcess{
		{PID: 72369, BuildCommit: "7606b7e"},
	}

	// Direct client PID (37111) only matches its own child MCP server (37128), not sibling session 72264's MCP server (72369).
	direct := FilterClientMCPProcesses(psTree, 37201, 37111, recorded, "unknown")
	if len(direct) != 1 || direct[0].PID != 37128 || direct[0].BuildCommit != "unknown" {
		t.Fatalf("FilterClientMCPProcesses direct clientPID = %+v, want [PID 37128 unknown]", direct)
	}

	// Shell wrapper PID (37200 /bin/sh) steps up to parent client 37111 and finds 37128.
	viaShell := FilterClientMCPProcesses(psTree, 37201, 37200, recorded, "unknown")
	if len(viaShell) != 1 || viaShell[0].PID != 37128 {
		t.Fatalf("FilterClientMCPProcesses via shell wrapper = %+v, want [PID 37128]", viaShell)
	}

	// Unrelated process (e.g. go test PID 90000) matches nothing.
	if unrelated := FilterClientMCPProcesses(psTree, 90001, 90000, recorded, "unknown"); len(unrelated) != 0 {
		t.Fatalf("expected 0 processes for unrelated clientPID, got %+v", unrelated)
	}

	// Format check: unknown and older build commits warn, matching HEAD does not.
	wUnknown := CheckClientMCPProcesses(direct, "990e19c")
	wantUnknown := "MCP server PID 37128 (build unknown) is older than HEAD; restart the client"
	if len(wUnknown) != 1 || wUnknown[0] != wantUnknown {
		t.Fatalf("CheckClientMCPProcesses unknown = %v, want [%q]", wUnknown, wantUnknown)
	}

	otherClient := FilterClientMCPProcesses(psTree, 99999, 72264, recorded, "unknown")
	wOlder := CheckClientMCPProcesses(otherClient, "990e19c")
	wantOlder := "MCP server PID 72369 (build 7606b7e) is older than HEAD; restart the client"
	if len(wOlder) != 1 || wOlder[0] != wantOlder {
		t.Fatalf("CheckClientMCPProcesses older = %v, want [%q]", wOlder, wantOlder)
	}

	if wCurrent := CheckClientMCPProcesses(otherClient, "7606b7e"); len(wCurrent) != 0 {
		t.Fatalf("expected 0 warnings when MCP build matches HEAD, got %v", wCurrent)
	}
}
