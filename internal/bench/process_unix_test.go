//go:build unix

package bench

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessGroupCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := filepath.Join(t.TempDir(), "ready")
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `sleep 30 & child=$!; echo "$child" > "$1"; wait`, "probe", path)
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	// Child PID is an explicit readiness signal; a bounded poll only waits for that signal.
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	var pid int
ready:
	for {
		select {
		case <-deadline:
			cancel()
			t.Fatal("child did not signal readiness")
		case <-tick.C:
			b, err := os.ReadFile(path)
			if err == nil {
				pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
				if pid > 0 {
					break ready
				}
			}
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("parent survived cancellation")
	}
	// Reaping can lag termination. A zombie is stopped work and cannot execute further.
	for until := time.Now().Add(3 * time.Second); time.Now().Before(until); {
		if syscall.Kill(pid, 0) == syscall.ESRCH {
			return
		}
		state, _ := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
		if strings.HasPrefix(strings.TrimSpace(string(state)), "Z") {
			return
		}
		<-tick.C
	}
	t.Fatal("descendant survived cancellation")
}
