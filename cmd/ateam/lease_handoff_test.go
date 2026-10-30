package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// writeLiveLease writes session A's lease, using this test process as A's live process.
func writeLiveLease(t *testing.T, root, agent, session string) []byte {
	t.Helper()
	for _, k := range []string{"ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CODEX_SESSION_ID", "GEMINI_SESSION_ID", "HERDR_ENV"} {
		t.Setenv(k, "")
	}
	t.Setenv("ATEAM_CLIENT_PID", strconv.Itoa(os.Getpid()))
	data, err := json.Marshal(state.WriterLease{Agent: agent, SessionID: session, PID: os.Getpid(), LastSeen: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.WriterLeasePath(root), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return data
}

// assertLeaseKept checks writer.json is byte-identical and the cached writer in state.json equals it.
func assertLeaseKept(t *testing.T, root string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(state.WriterLeasePath(root))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("writer.json = %s (%v), want byte-identical %s", got, err, want)
	}
	raw, err := os.ReadFile(state.CacheFilePath(root))
	if err != nil {
		t.Fatal(err)
	}
	var cache struct {
		Writer *state.WriterLease `json:"writer"`
	}
	var lease state.WriterLease
	if err := json.Unmarshal(raw, &cache); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &lease); err != nil {
		t.Fatal(err)
	}
	if cache.Writer == nil || !reflect.DeepEqual(*cache.Writer, lease) {
		t.Fatalf("cached writer = %+v, want %+v", cache.Writer, lease)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestStateHandoffKeepsAnotherSessionsWriterLease(t *testing.T) {
	// #265: session B's handoff deleted session A's live lease.
	for _, leaseAgent := range []string{"codex", "claude"} {
		t.Run("lease agent "+leaseAgent, func(t *testing.T) {
			dir := t.TempDir()
			want := writeLiveLease(t, dir, leaseAgent, "A")
			code, stderr := captureBenchError(t, func() int {
				return runStateHandoff(dir, []string{"--agent", "claude", "--session", "B", "--goal", "goal B", "--no-post"})
			})
			if code != 0 || stderr != "" {
				t.Fatalf("session B handoff = %d, stderr %q; want 0 and silent", code, stderr)
			}
			assertLeaseKept(t, dir, want)
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestStateHandoffReleasesOwnWriterLease(t *testing.T) {
	dir := t.TempDir()
	writeLiveLease(t, dir, "claude", "B")
	if code := runStateHandoff(dir, []string{"--agent", "claude", "--session", "B", "--goal", "goal B", "--no-post"}); code != 0 {
		t.Fatalf("own handoff = %d, want 0", code)
	}
	if _, err := os.Stat(state.WriterLeasePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("own writer.json still present (stat err %v)", err)
	}
	if st, _ := state.Load(dir); st.Writer != nil {
		t.Fatalf("cached writer = %+v, want none", st.Writer)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestStateHandoffReportsUnreleasedLease(t *testing.T) {
	dir := t.TempDir()
	writeLiveLease(t, dir, "claude", "B")
	t.Cleanup(func() { state.AfterLeaseDecision = nil })
	state.AfterLeaseDecision = func() {
		// A non-empty directory at writer.json cannot be removed.
		_ = os.Remove(state.WriterLeasePath(dir))
		if err := os.MkdirAll(filepath.Join(state.WriterLeasePath(dir), "x"), 0o755); err != nil {
			t.Error(err)
		}
	}
	code, stderr := captureBenchError(t, func() int {
		return runStateHandoff(dir, []string{"--agent", "claude", "--session", "B", "--goal", "goal B", "--no-post"})
	})
	if code != 1 || !strings.Contains(stderr, "ateam state handoff: handoff saved; writer lease not released: ") {
		t.Fatalf("handoff = %d, stderr %q; want 1 naming the saved handoff and unreleased lease", code, stderr)
	}
	if st, _ := state.Load(dir); st.Handoffs["claude@main"] == nil || st.Handoffs["claude@main"].Goal != "goal B" {
		t.Fatalf("handoff slot = %+v, want goal B saved", st.Handoffs["claude@main"])
	}
}
