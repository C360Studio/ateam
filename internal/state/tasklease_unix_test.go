//go:build unix

package state

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Optional external task lease display
func TestTaskLeaseFIFODoesNotBlock(t *testing.T) {
	clearAgentSessionEnv(t)
	t.Setenv("ATEAM_TASK_LEASE_FILE", "")
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, ".ateam", "task-lease.json"), 0o644); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	if got := pickupWithTaskLease(t, dir); !strings.Contains(got, "WARNING: task lease file unreadable (not a regular file)\n") {
		t.Fatalf("FIFO must warn:\n%s", got)
	}
}
