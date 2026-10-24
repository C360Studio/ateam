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

// fakeMCPInspection replaces the process lister, the registry loader and the cwd lookup behind
// InspectRunningMCPProcesses and InspectClientMCPProcesses. ps lists a client 100 and its child
// `ateam mcp` PID 41001; cwd counts the lookups.
func fakeMCPInspection(t *testing.T, recorded []state.MCPProcess, cwd string, cwdOK bool) *int {
	t.Helper()
	lookups := new(int)
	origList, origLoad, origCwd := listProcesses, loadLiveMCPProcesses, mcpProcessCwd
	listProcesses = func(context.Context) ([]byte, error) {
		return []byte("  100     1 Tue Oct 20 12:00:00 2026 /usr/local/bin/claude\n" +
			"41001   100 Tue Oct 20 12:00:01 2026 ateam mcp\n"), nil
	}
	loadLiveMCPProcesses = func(string) []state.MCPProcess { return recorded }
	mcpProcessCwd = func(_ context.Context, pid int) (string, bool) {
		*lookups++
		if pid != 41001 {
			t.Errorf("cwd lookup for PID %d, want 41001", pid)
		}
		return cwd, cwdOK
	}
	t.Cleanup(func() { listProcesses, loadLiveMCPProcesses, mcpProcessCwd = origList, origLoad, origCwd })
	return lookups
}

// spec: openspec/specs/harness/spec.md#Requirement: MCP process attribution
func TestMCPProcessAttribution(t *testing.T) {
	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	wt := filepath.Join(base, "main-wt")
	otherDir := filepath.Join(base, "other")
	writeLinkedWorktree(t, mainDir, wt)
	writeGoMod(t, mainDir, "module github.com/c360studio/ateam\n")
	writeGoMod(t, wt, "module github.com/c360studio/ateam\n")
	if err := os.MkdirAll(filepath.Join(otherDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	const head = "990e19c"
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
			var recorded []state.MCPProcess
			if tc.registered {
				recorded = []state.MCPProcess{{PID: 41001, BuildCommit: "7606b7e"}}
			}
			lookups := fakeMCPInspection(t, recorded, tc.cwd, tc.cwdOK)
			wantWarnings := 1
			if tc.wantOther {
				wantWarnings = 0
			}

			doctorProcs, doctorWarnings := InspectRunningMCPProcesses(context.Background(), tc.root, head)
			if len(doctorProcs) != 1 {
				t.Fatalf("doctor running_mcp_processes = %+v, want PID 41001", doctorProcs)
			}
			raw, err := json.Marshal(doctorProcs[0])
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(raw), `"repo":"other"`); got != tc.wantOther {
				t.Fatalf("doctor running_mcp_processes entry %s: repo other = %v, want %v", raw, got, tc.wantOther)
			}
			if len(doctorWarnings) != wantWarnings {
				t.Fatalf("doctor warnings %v, want %d", doctorWarnings, wantWarnings)
			}

			hookProcs, hookLines := InspectClientMCPProcesses(context.Background(), tc.root, head, 100)
			if len(hookProcs) != 1 {
				t.Fatalf("hook client-tree processes = %+v, want PID 41001", hookProcs)
			}
			if got := hookProcs[0].Repo == mcpRepoOther; got != tc.wantOther {
				t.Fatalf("hook process %+v: repo other = %v, want %v", hookProcs[0], got, tc.wantOther)
			}
			if len(hookLines) != wantWarnings {
				t.Fatalf("hook lines %v, want %d", hookLines, wantWarnings)
			}
			if wantLookups := map[bool]int{true: 0, false: 2}[tc.registered]; *lookups != wantLookups {
				t.Fatalf("cwd lookups = %d, want %d", *lookups, wantLookups)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: MCP process attribution
func TestDoctorComparesMCPBuildWithHeadOnlyInAteamCheckout(t *testing.T) {
	tests := []struct {
		name     string
		goMod    *string
		wantWarn bool
	}{
		{"ateam go.mod", ptr("module github.com/c360studio/ateam\n"), true},
		{"foreign go.mod", ptr("module example.com/adopter\n"), false},
		{"no go.mod", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.goMod != nil {
				writeGoMod(t, root, *tc.goMod)
			}
			// An unregistered server whose cwd is this root: ours, build unknown, and no bin/ateam on disk.
			fakeMCPInspection(t, nil, root, true)
			procs, warnings := InspectRunningMCPProcesses(context.Background(), root, "076a37f")
			if len(procs) != 1 || procs[0].Repo != "" {
				t.Fatalf("running_mcp_processes = %+v, want PID 41001 attributed to this root", procs)
			}
			if got := len(warnings) > 0; got != tc.wantWarn {
				t.Fatalf("doctor MCP warnings %v: warned = %v, want %v", warnings, got, tc.wantWarn)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: MCP process attribution
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
