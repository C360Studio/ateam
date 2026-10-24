package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

// writeLinkedWorktree lays out main/.git and a linked worktree wt the way `git worktree add` does,
// without running git.
func writeLinkedWorktree(t *testing.T, mainDir, wt string) {
	t.Helper()
	admin := filepath.Join(mainDir, ".git", "worktrees", "wt")
	for _, d := range []string{admin, wt} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(admin, "gitdir"):    filepath.Join(wt, ".git") + "\n",
		filepath.Join(admin, "commondir"): "../..\n",
		filepath.Join(wt, ".git"):         "gitdir: " + admin + "\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// spec: openspec/changes/mcp-attribution-151/specs/harness/spec.md#Requirement: MCP process attribution
func TestMCPProcessAttribution(t *testing.T) {
	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	wt := filepath.Join(base, "main-wt")
	otherDir := filepath.Join(base, "other")
	writeLinkedWorktree(t, mainDir, wt)
	if err := os.MkdirAll(filepath.Join(otherDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	const pid, head = 41001, "990e19c"
	tests := []struct {
		name       string
		root       string
		registered bool // in root's registry, i.e. in the recorded list LoadLiveMCPProcesses returns
		cwd        string
		cwdOK      bool
		wantOther  bool
	}{
		{"registered under another root, cwd there", mainDir, false, otherDir, true, true},
		{"unregistered, foreign cwd", mainDir, false, filepath.Join(otherDir, "sub"), true, true},
		{"sibling directory sharing the root prefix", mainDir, false, mainDir + "-extra", true, true},
		{"unregistered, cwd is this root", mainDir, false, mainDir, true, false},
		{"unregistered, cwd inside this root", mainDir, false, filepath.Join(mainDir, "internal"), true, false},
		{"unregistered, cwd is a linked worktree", mainDir, false, filepath.Join(wt, "cmd"), true, false},
		{"root is the worktree, cwd is the main checkout", wt, false, mainDir, true, false},
		{"registered here, foreign cwd", mainDir, true, otherDir, true, false},
		{"unregistered, unreadable cwd", mainDir, false, "", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookups := 0
			orig := mcpProcessCwd
			mcpProcessCwd = func(_ context.Context, got int) (string, bool) {
				lookups++
				if got != pid {
					t.Errorf("cwd lookup for PID %d, want %d", got, pid)
				}
				return tc.cwd, tc.cwdOK
			}
			t.Cleanup(func() { mcpProcessCwd = orig })

			var recorded []state.MCPProcess
			if tc.registered {
				recorded = []state.MCPProcess{{PID: pid, BuildCommit: "7606b7e"}}
			}
			// Fake process lister: `ps -eo pid=,ppid=,lstart=,args=` text with a client 100 and its MCP child.
			psOut := "  100     1 Tue Oct 20 12:00:00 2026 /usr/local/bin/claude\n" +
				"41001   100 Tue Oct 20 12:00:01 2026 ateam mcp\n"

			doctorProcs := ParseRunningMCPProcesses(psOut, 99999, recorded, "unknown")
			attributeMCPProcesses(context.Background(), tc.root, doctorProcs, recorded)
			hookProcs := FilterClientMCPProcesses(psOut, 99999, 100, recorded, "unknown")
			attributeMCPProcesses(context.Background(), tc.root, hookProcs, recorded)
			if len(doctorProcs) != 1 || len(hookProcs) != 1 {
				t.Fatalf("lister found doctor=%+v hook=%+v, want PID %d in both", doctorProcs, hookProcs, pid)
			}
			if wantLookups := map[bool]int{true: 0, false: 2}[tc.registered]; lookups != wantLookups {
				t.Fatalf("cwd lookups = %d, want %d", lookups, wantLookups)
			}

			raw, err := json.Marshal(doctorProcs[0])
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(raw), `"repo":"other"`); got != tc.wantOther {
				t.Fatalf("running_mcp_processes entry %s: repo other = %v, want %v", raw, got, tc.wantOther)
			}
			doctorWarnings := CheckRunningMCPProcesses(doctorProcs, head)
			hookWarnings := CheckClientMCPProcesses(hookProcs, head)
			wantWarnings := 1
			if tc.wantOther {
				wantWarnings = 0
			}
			if len(doctorWarnings) != wantWarnings || len(hookWarnings) != wantWarnings {
				t.Fatalf("doctor warnings %v, hook lines %v, want %d each", doctorWarnings, hookWarnings, wantWarnings)
			}
		})
	}
}

// spec: openspec/changes/mcp-attribution-151/specs/harness/spec.md#Requirement: MCP process attribution
func TestReadProcessCwdReadsOwnWorkingDirectory(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("working-directory lookup is implemented for darwin and linux only")
	}
	want, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := readProcessCwd(context.Background(), os.Getpid())
	if !ok {
		t.Fatalf("readProcessCwd(self) unreadable, want %s", want)
	}
	realGot, _ := filepath.EvalSymlinks(got)
	realWant, _ := filepath.EvalSymlinks(want)
	if realGot != realWant {
		t.Fatalf("readProcessCwd(self) = %q, want %q", got, want)
	}
	if _, ok := readProcessCwd(context.Background(), -1); ok {
		t.Fatal("readProcessCwd(-1) reported a working directory")
	}
}
