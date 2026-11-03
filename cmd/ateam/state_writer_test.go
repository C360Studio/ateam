package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

func writeRawWriterLease(t *testing.T, root, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.WriterLeasePath(root), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Machine-readable writer lease
func TestStateWriterJSON(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	t.Run("foreign live lease", func(t *testing.T) {
		// The daemon's own session differs from the lease's: it must still read live (#284).
		t.Setenv("ATEAM_SESSION_ID", "daemon")
		dir := t.TempDir()
		writeRawWriterLease(t, dir, `{"agent":"codex","session_id":"foreign","pid":`+strconv.Itoa(os.Getppid())+`,"last_seen":"`+now+`","herdr_pane_id":"p1"}`)
		code, out := captureStdout(t, func() int { return runState(dir, []string{"writer", "--json"}) })
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		if got["live"] != true || got["exited"] != false {
			t.Fatalf("live/exited = %v/%v, want true/false: %s", got["live"], got["exited"], out)
		}
		if got["agent"] != "codex" || got["session_id"] != "foreign" || got["herdr_pane_id"] != "p1" || got["last_seen"] != now {
			t.Fatalf("stored fields not printed: %s", out)
		}
	})
	t.Run("exited pid", func(t *testing.T) {
		dir := t.TempDir()
		writeRawWriterLease(t, dir, `{"agent":"codex","session_id":"s","pid":999999,"last_seen":"`+now+`"}`)
		code, out := captureStdout(t, func() int { return runState(dir, []string{"writer", "--json"}) })
		var got map[string]any
		if err := json.Unmarshal([]byte(out), &got); err != nil || code != 0 {
			t.Fatalf("exit %d, %v: %s", code, err, out)
		}
		if got["live"] != false || got["exited"] != true {
			t.Fatalf("live/exited = %v/%v, want false/true: %s", got["live"], got["exited"], out)
		}
	})
	t.Run("no lease", func(t *testing.T) {
		code, out := captureStdout(t, func() int { return runState(t.TempDir(), []string{"writer", "--json"}) })
		if code != 0 || strings.TrimSpace(out) != "null" {
			t.Fatalf("exit %d, output %q; want 0 and null", code, out)
		}
	})
	t.Run("corrupt lease is not null", func(t *testing.T) {
		dir := t.TempDir()
		writeRawWriterLease(t, dir, "{")
		code, stderr := captureBenchError(t, func() int { return runState(dir, []string{"writer", "--json"}) })
		if code != 1 || !strings.Contains(stderr, "ateam state writer:") {
			t.Fatalf("exit %d, stderr %q; want 1", code, stderr)
		}
	})
	for _, args := range [][]string{{"writer"}, {"writer", "--json", "extra"}, {"writer", "--nope"}} {
		t.Run("usage "+strings.Join(args, " "), func(t *testing.T) {
			code, _ := captureBenchError(t, func() int { return runState(t.TempDir(), args) })
			if code != 2 {
				t.Fatalf("exit %d, want 2", code)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional external task lease display
func TestStatePickupPrintsTaskLeaseThroughRefresh(t *testing.T) {
	// Refresh goes through Update, not a plain Load: the json:"-" fields must survive to the summary.
	t.Setenv("ATEAM_TASK_LEASE_FILE", "")
	t.Setenv("ATEAM_FORGE", "none")
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	lease := `{"id":"abc","issue":7,"holder":"h","expires_at":"2099-01-02T03:04:05Z","source":"s"}`
	if err := os.WriteFile(filepath.Join(dir, ".ateam", "task-lease.json"), []byte(lease), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := captureStdout(t, func() int { return runState(dir, []string{"pickup"}) })
	if code != 0 || !strings.Contains(out, "Task lease: issue #7 holder=h") {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
}
