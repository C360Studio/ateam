package bootstrap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// spec: harness / Doctor checks the verify target
func TestCheckVerifyTarget(t *testing.T) {
	const (
		missing  = "Taskfile.yml: missing; the shipped skills and protocol run task verify"
		noTarget = "Taskfile.yml: no verify target; the shipped skills and protocol run task verify"
	)
	cases := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{name: "no Taskfile", want: []string{missing}},
		// The semsource shape from #415: a check task and no verify.
		{name: "check only", files: map[string]string{"Taskfile.yml": "version: '3'\ntasks:\n  check:\n    cmds:\n      - go test ./...\n"}, want: []string{noTarget}},
		{name: "verify present", files: map[string]string{"Taskfile.yml": "version: '3'\ntasks:\n  check:\n    cmds:\n      - go test ./...\n  verify:\n    cmds:\n      - task: check\n"}},
		// Owner ruling D3 A on PR #436: an alias is not a task `ateam task verify` accepts.
		{name: "verify only as an alias", files: map[string]string{"Taskfile.yml": "version: '3'\ntasks:\n  check:\n    aliases: [verify]\n    cmds:\n      - go test ./...\n"}, want: []string{noTarget}},
		// verify nested under another top-level key is not a task.
		{name: "verify outside tasks", files: map[string]string{"Taskfile.yml": "version: '3'\nvars:\n  verify: x\ntasks:\n  check:\n    cmds:\n      - go test ./...\n"}, want: []string{noTarget}},
	}
	run := func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				dir := t.TempDir()
				for rel, body := range c.files {
					if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				if got := CheckVerifyTarget(context.Background(), dir); !slices.Equal(got, c.want) {
					t.Fatalf("CheckVerifyTarget = %q, want %q", got, c.want)
				}
			})
		}
	}
	// Without the Task executable ListTargets scans the file; with it, Task lists the tasks. Both must agree.
	t.Run("file scan", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		run(t)
	})
	t.Run("task CLI", func(t *testing.T) {
		if _, err := exec.LookPath("task"); err != nil {
			t.Skip("task executable not on PATH")
		}
		run(t)
	})
}

// spec: harness / Doctor checks the verify target
func TestCheckVerifyTargetFlattenedInclude(t *testing.T) {
	if _, err := exec.LookPath("task"); err != nil {
		t.Skip("task executable not on PATH: only Task resolves includes")
	}
	dir := t.TempDir()
	files := map[string]string{
		"Taskfile.yml": "version: '3'\nincludes:\n  ci:\n    taskfile: ./ci.yml\n    flatten: true\ntasks:\n  check:\n    cmds:\n      - go test ./...\n",
		"ci.yml":       "version: '3'\ntasks:\n  verify:\n    cmds:\n      - echo ok\n",
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := CheckVerifyTarget(context.Background(), dir); got != nil {
		t.Fatalf("CheckVerifyTarget = %q, want no violation for a verify task from a flattened include", got)
	}
}
