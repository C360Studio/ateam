//go:build unix

package web

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
func TestModelRoutingReplacedFIFO(t *testing.T) {
	dir := t.TempDir()
	path := writeRoutingPolicy(t, dir, routingPolicyFixture)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	// Reproduce the exact replacement boundary: Lstat saw a regular file, but
	// the subsequent descriptor open sees a FIFO with no writer.
	info, err := root.Lstat(".agents/review-policy.json")
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("precondition: %v, %v", info, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := readRoutingFile(root, filepath.Join(".agents", "review-policy.json")); done <- err }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("expected descriptor rejection, got %v", err)
		}
	case <-time.After(2 * time.Second):
		// Unblock an incorrect blocking open so this regression never leaves a worker.
		writer, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			writer.Close()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("FIFO reader failed to terminate after cleanup")
		}
		t.Fatal("policy replacement blocked on a FIFO")
	}
}
