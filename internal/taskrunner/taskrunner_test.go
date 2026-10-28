package taskrunner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
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
	targets, err := ListTargets(context.Background(), dir)
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
	// The fake answers --list-all --json as Task 3.51 does for an includes: layout, where a label: makes the display
	// name differ from the task key, and echoes any target it is asked to run. Included and labeled targets must be
	// listed by key and accepted by Run.
	dir := writeTaskfile(t, includesTaskfile)
	writeFakeTask(t, `[ "$1 $2" = "--taskfile Taskfile.yml" ] || { echo "unpinned taskfile: $*" >&2; exit 2; }
if [ "$3" = "--list-all" ]; then
  echo '{"tasks":[{"name":"test:unit","task":"test:unit","desc":"Run unit tests"},{"name":"Format code","task":"fmt","desc":"Format Go code"}]}'
  exit 0
fi
echo "ran $3"
`)
	targets, err := ListTargets(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	want := []Target{{Name: "fmt", Desc: "Format Go code"}, {Name: "test:unit", Desc: "Run unit tests"}}
	if len(targets) != len(want) || targets[0] != want[0] || targets[1] != want[1] {
		t.Fatalf("targets = %+v, want %+v", targets, want)
	}
	for _, target := range []string{"test:unit", "fmt"} {
		out, err := Run(context.Background(), dir, target, nil)
		if err != nil || strings.TrimSpace(out) != "ran "+target {
			t.Fatalf("Run(%s) = %q, %v", target, out, err)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestListTargetsRequiresRootTaskfile(t *testing.T) {
	// Task would find a Taskfile in an ancestor directory; without root's own Taskfile.yml nothing may be invoked.
	parent := writeTaskfile(t, includesTaskfile)
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "invoked")
	writeFakeTask(t, ": > "+marker+"\necho '{\"tasks\":[{\"name\":\"fmt\",\"task\":\"fmt\"}]}'\n")
	if _, err := Run(context.Background(), child, "fmt", nil); err == nil || !strings.Contains(err.Error(), "read Taskfile.yml") {
		t.Fatalf("Run in a child without Taskfile.yml: err = %v, want read Taskfile.yml error", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("task was invoked for a root without its own Taskfile.yml")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestRunSkipsDiscoveryOnCanceledContext(t *testing.T) {
	dir := writeTaskfile(t, includesTaskfile)
	marker := filepath.Join(t.TempDir(), "invoked")
	writeFakeTask(t, ": > "+marker+"\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, dir, "fmt", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run on canceled ctx: err = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("task was invoked despite a canceled context")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestListTargetsStopsOnCancelDespiteHeldPipe(t *testing.T) {
	// The fake leaves a descendant holding stdout, signals through a FIFO that listing is in flight, then blocks.
	// Cancellation must end listing with ctx's error (no fallback) without waiting for the descendant.
	dir := writeTaskfile(t, includesTaskfile)
	fifo := filepath.Join(t.TempDir(), "started")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	writeFakeTask(t, "/bin/sleep 30 &\necho started > "+fifo+"\nwait\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		targets []Target
		err     error
	}
	done := make(chan result, 1)
	go func() {
		targets, err := ListTargets(ctx, dir)
		done <- result{targets, err}
	}()
	if _, err := os.ReadFile(fifo); err != nil { // blocks until the fake has started
		t.Fatal(err)
	}
	canceled := time.Now()
	cancel()
	select {
	case r := <-done:
		if !errors.Is(r.err, context.Canceled) {
			t.Fatalf("ListTargets after cancel = %+v, %v; want context.Canceled", r.targets, r.err)
		}
		if waited := time.Since(canceled); waited > 10*time.Second {
			t.Fatalf("listing returned %v after cancel; the held pipe was waited on", waited)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("listing still blocked 20s after cancel; the held pipe was waited on")
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
			targets, err := ListTargets(context.Background(), dir)
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
