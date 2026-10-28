package taskrunner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestListTargetsAndValidation(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no task CLI: the root Taskfile.yml scan is the fallback
	dir := t.TempDir()
	tf := `version: '3'
tasks:
  fmt:
    desc: Format Go code
    cmds:
      - gofmt -w .
  verify:
    desc: Run all checks
    cmds:
      - go test ./...
`
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(tf), 0o644); err != nil {
		t.Fatal(err)
	}
	targets, err := ListTargets(dir)
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(targets) != 2 || targets[0].Name != "fmt" || targets[1].Name != "verify" {
		t.Fatalf("unexpected targets: %+v", targets)
	}

	_, err = Run(context.Background(), dir, "nonexistent", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown task target") {
		t.Fatalf("expected unknown task target error, got: %v", err)
	}
}

const includesTaskfile = `version: '3'
includes:
  test: ./taskfiles/test.yml
tasks:
  fmt:
    desc: Format Go code
`

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestListTargetsResolvesIncludesWithTaskCLI(t *testing.T) {
	// The fake answers --list-all --json as Task v3 does for an includes: layout, and echoes any target it is asked
	// to run, so an included target must be both listed and accepted by Run.
	dir := writeTaskfile(t, includesTaskfile)
	writeFakeTask(t, `if [ "$1" = "--list-all" ]; then
  echo '{"tasks":[{"name":"test:unit","task":"test:unit","desc":"Run unit tests"},{"name":"fmt","task":"fmt","desc":"Format Go code"}]}'
  exit 0
fi
echo "ran $1"
`)
	targets, err := ListTargets(dir)
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	want := []Target{{Name: "fmt", Desc: "Format Go code"}, {Name: "test:unit", Desc: "Run unit tests"}}
	if len(targets) != len(want) || targets[0] != want[0] || targets[1] != want[1] {
		t.Fatalf("targets = %+v, want %+v", targets, want)
	}
	out, err := Run(context.Background(), dir, "test:unit", nil)
	if err != nil || strings.TrimSpace(out) != "ran test:unit" {
		t.Fatalf("Run(test:unit) = %q, %v", out, err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestListTargetsFallsBackWhenTaskCLIFails(t *testing.T) {
	for name, body := range map[string]string{
		// Valid JSON on a failed exit must still be distrusted, so only the exit status triggers the fallback.
		"exit status": "echo '{\"tasks\":[{\"name\":\"stale\"}]}'\nexit 1\n",
		"bad json":    "echo 'not json'\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeTaskfile(t, includesTaskfile)
			writeFakeTask(t, body)
			targets, err := ListTargets(dir)
			if err != nil {
				t.Fatalf("ListTargets: %v", err)
			}
			if len(targets) != 1 || targets[0] != (Target{Name: "fmt", Desc: "Format Go code"}) {
				t.Fatalf("targets = %+v, want only the scanned fmt target", targets)
			}
		})
	}
}

func writeTaskfile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeFakeTask puts a shell fake named task alone on PATH and execs it once outside any deadline: the first exec of
// a new file on macOS can be slow.
func writeFakeTask(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	path := filepath.Join(bin, "task")
	script := "#!/bin/sh\n[ -n \"$ATEAM_FAKE_PREWARM\" ] && exit 0\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), "ATEAM_FAKE_PREWARM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prewarm %s: %v\n%s", path, err, out)
	}
	t.Setenv("PATH", bin)
}
