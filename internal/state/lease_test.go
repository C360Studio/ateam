package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// clearLeaseIdentityEnv pins the caller identity sources so a developer's session env cannot decide ownership.
func clearLeaseIdentityEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CODEX_SESSION_ID", "GEMINI_SESSION_ID", "HERDR_ENV", "HERDR_PANE_ID"} {
		t.Setenv(k, "")
	}
	t.Setenv("ATEAM_CLIENT_PID", strconv.Itoa(os.Getpid()))
}

func writeTestLease(t *testing.T, root string, wl WriterLease) []byte {
	t.Helper()
	data, err := json.Marshal(wl)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(WriterLeasePath(root), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return data
}

func testHandoff(session string) Handoff {
	return Handoff{Goal: "goal " + session, Agent: "claude", SessionID: session, Branch: "main", UpdatedAt: time.Now().UTC()}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestOwnsWriterLease(t *testing.T) {
	clearLeaseIdentityEnv(t)
	now := time.Now().UTC()
	live := os.Getpid()
	foreignLive := os.Getppid()
	tests := []struct {
		name   string
		lease  *WriterLease
		caller string
		herdr  string
		want   bool
	}{
		{"same session", &WriterLease{Agent: "claude", SessionID: "A", PID: foreignLive, LastSeen: now}, "A", "", true},
		{"different session same agent", &WriterLease{Agent: "claude", SessionID: "A", PID: live, LastSeen: now}, "B", "", false},
		{"different agent", &WriterLease{Agent: "codex", SessionID: "A", PID: live, LastSeen: now}, "B", "", false},
		{"caller session against lease without one", &WriterLease{Agent: "claude", PID: live, LastSeen: now}, "B", "", false},
		{"no caller session, own PID", &WriterLease{Agent: "claude", SessionID: "A", PID: live, LastSeen: now}, "", "", true},
		{"no caller session, foreign live PID", &WriterLease{Agent: "claude", SessionID: "A", PID: foreignLive, LastSeen: now}, "", "", false},
		{"no caller session, matching herdr pane", &WriterLease{Agent: "claude", SessionID: "A", PID: foreignLive, LastSeen: now, HerdrPaneID: "p1"}, "", "p1", true},
		{"no caller session, other herdr pane", &WriterLease{Agent: "claude", SessionID: "A", PID: foreignLive, LastSeen: now, HerdrPaneID: "p2"}, "", "p1", false},
		{"dead PID", &WriterLease{Agent: "claude", SessionID: "A", PID: 999999, LastSeen: now}, "A", "", false},
		{"stale last_seen", &WriterLease{Agent: "claude", SessionID: "A", PID: live, LastSeen: now.Add(-11 * time.Minute)}, "A", "", false},
		{"lease with no identity", &WriterLease{Agent: "claude", LastSeen: now}, "", "", false},
		{"no lease", nil, "A", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.herdr != "" {
				t.Setenv("HERDR_ENV", "1")
				t.Setenv("HERDR_PANE_ID", tc.herdr)
			}
			if got := ownsWriterLease(tc.lease, tc.caller); got != tc.want {
				t.Fatalf("ownsWriterLease(%+v, %q) = %v, want %v", tc.lease, tc.caller, got, tc.want)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestSaveHandoffReleasingLease(t *testing.T) {
	clearLeaseIdentityEnv(t)
	t.Run("own lease is released", func(t *testing.T) {
		root := t.TempDir()
		writeTestLease(t, root, WriterLease{Agent: "claude", SessionID: "A", PID: os.Getpid(), LastSeen: time.Now().UTC()})
		kept, err := SaveHandoffReleasingLease(root, testHandoff("A"), false, "A", nil)
		if err != nil || kept != nil {
			t.Fatalf("SaveHandoffReleasingLease = (%+v, %v), want (nil, nil)", kept, err)
		}
		if _, err := os.Stat(WriterLeasePath(root)); !os.IsNotExist(err) {
			t.Fatalf("writer.json still present (stat err %v)", err)
		}
		st, err := Load(root)
		if err != nil || st.Writer != nil || st.Handoffs["claude@main"] == nil {
			t.Fatalf("Load = (writer %+v, slot %+v, %v), want no writer and a saved slot", st.Writer, st.Handoffs["claude@main"], err)
		}
	})
	t.Run("foreign lease is kept", func(t *testing.T) {
		root := t.TempDir()
		want := writeTestLease(t, root, WriterLease{Agent: "codex", SessionID: "A", PID: os.Getpid(), LastSeen: time.Now().UTC()})
		kept, err := SaveHandoffReleasingLease(root, testHandoff("B"), false, "B", nil)
		if err != nil || kept == nil || kept.SessionID != "A" {
			t.Fatalf("SaveHandoffReleasingLease = (%+v, %v), want the kept lease of session A", kept, err)
		}
		if got, _ := os.ReadFile(WriterLeasePath(root)); !bytes.Equal(got, want) {
			t.Fatalf("writer.json = %s, want unchanged %s", got, want)
		}
	})
	t.Run("refused handoff changes neither file", func(t *testing.T) {
		root := t.TempDir()
		if _, err := SaveHandoffReleasingLease(root, testHandoff("A"), false, "A", nil); err != nil {
			t.Fatalf("seed handoff A: %v", err)
		}
		lease := writeTestLease(t, root, WriterLease{Agent: "claude", SessionID: "B", PID: os.Getpid(), LastSeen: time.Now().UTC()})
		cache, err := os.ReadFile(CacheFilePath(root))
		if err != nil {
			t.Fatal(err)
		}
		_, err = SaveHandoffReleasingLease(root, testHandoff("B"), false, "B", nil)
		var held *HandoffSlotHeldError
		if !errors.As(err, &held) {
			t.Fatalf("handoff B over A's fresh slot err = %v, want *HandoffSlotHeldError", err)
		}
		if got, _ := os.ReadFile(WriterLeasePath(root)); !bytes.Equal(got, lease) {
			t.Fatalf("refused handoff changed writer.json: %s, want %s", got, lease)
		}
		if got, _ := os.ReadFile(CacheFilePath(root)); !bytes.Equal(got, cache) {
			t.Fatalf("refused handoff changed state.json")
		}
	})
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestLeaseWrittenDuringHandoffSurvives(t *testing.T) {
	clearLeaseIdentityEnv(t)
	root := t.TempDir()
	writeTestLease(t, root, WriterLease{Agent: "claude", SessionID: "A", PID: os.Getpid(), LastSeen: time.Now().UTC()})
	done := make(chan struct{})
	var writeErr error
	t.Cleanup(func() { AfterLeaseDecision, beforeFlock = nil, nil })
	AfterLeaseDecision = func() {
		// The handoff holds the lock and has judged A's lease as its own. A replacement write starts now and
		// either reaches the lock (and must wait) or finishes without it.
		blocking := make(chan struct{}, 1)
		beforeFlock = func() { blocking <- struct{}{} }
		go func() {
			defer close(done)
			_, writeErr = WriteWriterLease(root, "codex", "C")
		}()
		select {
		case <-blocking:
		case <-done:
		}
	}
	if _, err := SaveHandoffReleasingLease(root, testHandoff("A"), false, "A", nil); err != nil {
		t.Fatalf("SaveHandoffReleasingLease: %v", err)
	}
	<-done
	if writeErr != nil {
		t.Fatalf("WriteWriterLease: %v", writeErr)
	}
	wl, err := ReadWriterLease(root)
	if err != nil || wl == nil || wl.SessionID != "C" {
		t.Fatalf("lease after handoff = (%+v, %v), want session C's replacement lease", wl, err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestSaveHandoffReportsFailedRelease(t *testing.T) {
	clearLeaseIdentityEnv(t)
	root := t.TempDir()
	writeTestLease(t, root, WriterLease{Agent: "claude", SessionID: "A", PID: os.Getpid(), LastSeen: time.Now().UTC()})
	t.Cleanup(func() { AfterLeaseDecision = nil })
	AfterLeaseDecision = func() { blockLeaseRemoval(t, root) }
	_, err := SaveHandoffReleasingLease(root, testHandoff("A"), false, "A", nil)
	var rel *LeaseReleaseError
	if !errors.As(err, &rel) {
		t.Fatalf("err = %v, want *LeaseReleaseError", err)
	}
	st, err := Load(root)
	if err != nil || st.Handoffs["claude@main"] == nil || st.Handoffs["claude@main"].Goal != "goal A" {
		t.Fatalf("handoff not saved: %+v (%v)", st.Handoffs, err)
	}
}

// blockLeaseRemoval replaces writer.json with a non-empty directory, which os.Remove cannot delete.
func blockLeaseRemoval(t *testing.T, root string) {
	if err := os.Remove(WriterLeasePath(root)); err != nil {
		t.Error(err)
	}
	if err := os.MkdirAll(filepath.Join(WriterLeasePath(root), "x"), 0o755); err != nil {
		t.Error(err)
	}
}
