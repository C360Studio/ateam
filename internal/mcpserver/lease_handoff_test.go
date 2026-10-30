package mcpserver

import (
	"bytes"
	"context"
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

func handoffArgs(session string) json.RawMessage {
	return json.RawMessage(`{"action":"handoff","agent":"claude","session_id":"` + session + `","goal":"goal ` + session + `","post":false}`)
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestMCPHandoffKeepsAnotherSessionsWriterLease(t *testing.T) {
	// #265: session B's handoff deleted session A's live lease.
	for _, leaseAgent := range []string{"codex", "claude"} {
		t.Run("lease agent "+leaseAgent, func(t *testing.T) {
			dir := t.TempDir()
			want := writeLiveLease(t, dir, leaseAgent, "A")
			text, isErr := execStateTool(context.Background(), dir, handoffArgs("B"), "", "test")
			if isErr || !strings.Contains(text, "writer lease: kept (held by "+leaseAgent+" session A)") {
				t.Fatalf("handoff = (%q, isError %v), want success naming the kept lease", text, isErr)
			}
			if got, err := os.ReadFile(state.WriterLeasePath(dir)); err != nil || !bytes.Equal(got, want) {
				t.Fatalf("writer.json = %s (%v), want byte-identical %s", got, err, want)
			}
			raw, err := os.ReadFile(state.CacheFilePath(dir))
			if err != nil {
				t.Fatal(err)
			}
			var cache struct {
				Writer *state.WriterLease `json:"writer"`
			}
			var lease state.WriterLease
			_ = json.Unmarshal(raw, &cache)
			_ = json.Unmarshal(want, &lease)
			if cache.Writer == nil || !reflect.DeepEqual(*cache.Writer, lease) {
				t.Fatalf("cached writer = %+v, want %+v", cache.Writer, lease)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestMCPHandoffReleasesOwnWriterLease(t *testing.T) {
	dir := t.TempDir()
	writeLiveLease(t, dir, "claude", "B")
	text, isErr := execStateTool(context.Background(), dir, handoffArgs("B"), "", "test")
	if isErr || strings.Contains(text, "writer lease: kept") {
		t.Fatalf("own handoff = (%q, isError %v), want success without a kept lease", text, isErr)
	}
	if _, err := os.Stat(state.WriterLeasePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("own writer.json still present (stat err %v)", err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Handoff releases only the caller's writer lease
func TestMCPHandoffReportsUnreleasedLease(t *testing.T) {
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
	text, isErr := execStateTool(context.Background(), dir, handoffArgs("B"), "", "test")
	if !isErr || !strings.HasPrefix(text, "handoff saved; writer lease not released: ") {
		t.Fatalf("handoff = (%q, isError %v), want an error naming the saved handoff", text, isErr)
	}
	if st, _ := state.Load(dir); st.Handoffs["claude@main"] == nil || st.Handoffs["claude@main"].Goal != "goal B" {
		t.Fatalf("handoff slot = %+v, want goal B saved", st.Handoffs["claude@main"])
	}
}
