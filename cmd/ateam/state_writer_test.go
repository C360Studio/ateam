package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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

// spec: openspec/specs/harness/spec.md#Requirement: Machine-readable writer state
func TestStateWriterJSON(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	writerJSON := func(t *testing.T, dir string) map[string]any {
		t.Helper()
		code, out := captureStdout(t, func() int { return runState(dir, []string{"writer", "--json"}) })
		var got map[string]any
		if err := json.Unmarshal([]byte(out), &got); err != nil || code != 0 {
			t.Fatalf("exit %d, %v: %s", code, err, out)
		}
		return got
	}
	t.Run("foreign live lease written by this binary", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("HERDR_ENV", "1")
		t.Setenv("HERDR_PANE_ID", "p1")
		lease, err := state.WriteWriterLease(dir, "codex", "foreign")
		if err != nil {
			t.Fatal(err)
		}
		// The caller's own session differs from the lease's: it must still read live (#284).
		t.Setenv("ATEAM_SESSION_ID", "daemon")
		got := writerJSON(t, dir)
		if got["state"] != "live" || got["reason"] != "" {
			t.Fatalf("state/reason = %v/%v, want live and empty", got["state"], got["reason"])
		}
		if got["agent"] != "codex" || got["session_id"] != "foreign" || got["herdr_pane_id"] != "p1" || got["host"] != lease.Host {
			t.Fatalf("stored fields not printed: %v", got)
		}
	})
	t.Run("live whichever session holds it", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("ATEAM_SESSION_ID", "mine")
		if _, err := state.WriteWriterLease(dir, "codex", "mine"); err != nil {
			t.Fatal(err)
		}
		if got := writerJSON(t, dir); got["state"] != "live" {
			t.Fatalf("own session: got %v, want live", got)
		}
		if _, err := state.WriteWriterLease(dir, "codex", ""); err != nil {
			t.Fatal(err)
		}
		if got := writerJSON(t, dir); got["state"] != "live" {
			t.Fatalf("no session: got %v, want live", got)
		}
	})
	t.Run("lease from another host", func(t *testing.T) {
		dir := t.TempDir()
		writeRawWriterLease(t, dir, `{"agent":"codex","session_id":"s","pid":999999,"host":"elsewhere.invalid","last_seen":"`+now+`"}`)
		got := writerJSON(t, dir)
		if got["state"] != "unknown" || got["reason"] != "other host" || got["host"] != "elsewhere.invalid" {
			t.Fatalf("got %v, want unknown/other host", got)
		}
	})
	t.Run("no lease", func(t *testing.T) {
		got := writerJSON(t, t.TempDir())
		if got["state"] != "none" || got["reason"] != "" || len(got) != 2 {
			t.Fatalf("got %v, want only state none and empty reason", got)
		}
	})
	t.Run("corrupt lease exits 0 as unknown", func(t *testing.T) {
		dir := t.TempDir()
		writeRawWriterLease(t, dir, "{")
		got := writerJSON(t, dir)
		if got["state"] != "unknown" || got["reason"] != "lease unreadable" {
			t.Fatalf("got %v, want unknown/lease unreadable", got)
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

// spec: openspec/specs/harness/spec.md#Requirement: Machine-readable version and capabilities
func TestVersionJSON(t *testing.T) {
	code, out := captureStdout(t, func() int { return run([]string{"version", "--json"}) })
	var got struct {
		Version      string   `json:"version"`
		Commit       string   `json:"commit"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || code != 0 {
		t.Fatalf("exit %d, %v: %s", code, err, out)
	}
	if got.Version == "" || got.Commit == "" {
		t.Fatalf("missing version or commit: %s", out)
	}
	if !slices.Contains(got.Capabilities, "state-writer-json") || !slices.Contains(got.Capabilities, "task-lease-file") {
		t.Fatalf("capabilities = %v", got.Capabilities)
	}
	code, out = captureStdout(t, func() int { return run([]string{"version"}) })
	if code != 0 || !strings.HasPrefix(out, "ateam ") || json.Valid([]byte(out)) {
		t.Fatalf("plain version changed: exit %d, %q", code, out)
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
