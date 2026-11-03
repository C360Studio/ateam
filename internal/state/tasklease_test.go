package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validTaskLease = `{"id":"abc","issue":284,"holder":"worker-1","expires_at":"2099-01-02T03:04:05Z","source":"factory","extra":1}`

func writeTaskLease(t *testing.T, root, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".ateam"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ateam", "task-lease.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func pickupWithTaskLease(t *testing.T, root string) string {
	t.Helper()
	st, err := Load(root)
	if err != nil {
		t.Fatalf("Load must not fail on a task lease file: %v", err)
	}
	return BuildPickupSummary(st)
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional external task lease display
func TestTaskLeaseLine(t *testing.T) {
	clearAgentSessionEnv(t)
	t.Setenv("ATEAM_TASK_LEASE_FILE", "")

	t.Run("fresh", func(t *testing.T) {
		dir := t.TempDir()
		writeTaskLease(t, dir, validTaskLease)
		got := pickupWithTaskLease(t, dir)
		want := "Task lease: issue #284 holder=worker-1 expires=2099-01-02T03:04:05Z (source factory)\n"
		if !strings.Contains(got, want) {
			t.Fatalf("pickup missing %q:\n%s", want, got)
		}
	})
	t.Run("expired", func(t *testing.T) {
		dir := t.TempDir()
		writeTaskLease(t, dir, strings.Replace(validTaskLease, "2099", "2001", 1))
		got := pickupWithTaskLease(t, dir)
		want := "Task lease: issue #284 holder=worker-1 expires=2001-01-02T03:04:05Z (source factory) expired\n"
		if !strings.Contains(got, want) {
			t.Fatalf("pickup missing %q:\n%s", want, got)
		}
	})
	t.Run("absent prints nothing", func(t *testing.T) {
		got := pickupWithTaskLease(t, t.TempDir())
		if strings.Contains(strings.ToLower(got), "task lease") {
			t.Fatalf("absent file must print nothing:\n%s", got)
		}
	})
	t.Run("override paths", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "elsewhere.json"), []byte(validTaskLease), 0o644); err != nil {
			t.Fatal(err)
		}
		for name, p := range map[string]string{"relative": "elsewhere.json", "absolute": filepath.Join(dir, "elsewhere.json")} {
			t.Setenv("ATEAM_TASK_LEASE_FILE", p)
			if got := pickupWithTaskLease(t, dir); !strings.Contains(got, "Task lease: issue #284") {
				t.Fatalf("%s override not read:\n%s", name, got)
			}
		}
	})
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional external task lease display
func TestTaskLeaseMalformedWarnsOnce(t *testing.T) {
	clearAgentSessionEnv(t)
	t.Setenv("ATEAM_TASK_LEASE_FILE", "")
	secret := "SECRETVALUE"
	field := func(name, value string) string {
		var m map[string]any
		if err := json.Unmarshal([]byte(validTaskLease), &m); err != nil {
			t.Fatal(err)
		}
		m[name] = value
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, tc := range []struct{ name, content, reason string }{
		{"not JSON", "{" + secret, "invalid JSON or field type"},
		{"JSON array", `[1]`, "invalid JSON or field type"},
		{"issue string", strings.Replace(validTaskLease, "284", `"284"`, 1), "invalid JSON or field type"},
		{"issue fraction", strings.Replace(validTaskLease, "284", "28.4", 1), "invalid JSON or field type"},
		{"issue zero", strings.Replace(validTaskLease, "284", "0", 1), "issue must be a positive integer"},
		{"JSON null", `null`, "missing id"},
		{"missing holder", field("holder", ""), "missing holder"},
		{"missing source", field("source", ""), "missing source"},
		{"unparsable expiry", field("expires_at", secret), "expires_at is not RFC 3339"},
		{"newline in holder", field("holder", "a\nWARNING: "+secret), "holder has a control character"},
		{"line separator in id", field("id", "a\u2028"+secret), "id has a control character"},
		{"long holder", field("holder", strings.Repeat("x", taskLeaseMaxFieldBytes+1)), "holder too long"},
		{"oversized file", validTaskLease + strings.Repeat(" ", taskLeaseMaxFileBytes), "file too large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTaskLease(t, dir, tc.content)
			got := pickupWithTaskLease(t, dir)
			want := "WARNING: task lease file unreadable (" + tc.reason + ")\n"
			if strings.Count(got, want) != 1 {
				t.Fatalf("want one %q:\n%s", want, got)
			}
			if strings.Contains(got, "Task lease:") || strings.Contains(got, secret) {
				t.Fatalf("bad file leaked into pickup:\n%s", got)
			}
		})
	}
	t.Run("directory at the path", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".ateam", "task-lease.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := pickupWithTaskLease(t, dir); !strings.Contains(got, "WARNING: task lease file unreadable (not a regular file)\n") {
			t.Fatalf("directory must warn:\n%s", got)
		}
	})
	t.Run("boundary sizes accepted", func(t *testing.T) {
		dir := t.TempDir()
		content := field("holder", strings.Repeat("x", taskLeaseMaxFieldBytes))
		writeTaskLease(t, dir, content+strings.Repeat(" ", taskLeaseMaxFileBytes-len(content)))
		if got := pickupWithTaskLease(t, dir); !strings.Contains(got, "Task lease: issue #284") {
			t.Fatalf("file and field exactly at the limits must parse:\n%s", got)
		}
	})
}

// spec: openspec/specs/harness/spec.md#Requirement: Task lease token in handoff Addresses
func TestTaskLeaseAddressesToken(t *testing.T) {
	clearAgentSessionEnv(t)
	t.Setenv("ATEAM_TASK_LEASE_FILE", "")
	const mismatch = "handoff lease def is not the current task lease abc\n"
	for _, tc := range []struct {
		name, addresses string
		noFile          bool
		want            string
	}{
		{name: "mismatch", addresses: "PR #1, aip-lease:def, ruling", want: mismatch},
		{name: "case-insensitive prefix", addresses: "AIP-LEASE:def", want: mismatch},
		{name: "match", addresses: "aip-lease:abc"},
		{name: "id compared exactly, case differs", addresses: "aip-lease:ABC", want: "handoff lease ABC is not the current task lease abc\n"},
		{name: "longer id is not a match", addresses: "aip-lease:abcd", want: "handoff lease abcd is not the current task lease abc\n"},
		{name: "embedded in a word", addresses: "xaip-lease:def"},
		{name: "no token", addresses: "PR #1"},
		{name: "no file", addresses: "aip-lease:def", noFile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if !tc.noFile {
				writeTaskLease(t, dir, validTaskLease)
			}
			st, err := Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			st.CurrentAgent = "claude"
			st.Git.Branch = "claude/x"
			st.Handoffs = map[string]*Handoff{HandoffSlotKey("claude", "claude/x"): {Goal: "g", Addresses: tc.addresses}}
			got := BuildPickupSummary(st)
			if tc.want == "" && strings.Contains(got, "handoff lease ") {
				t.Fatalf("unexpected mismatch line:\n%s", got)
			}
			if tc.want != "" && strings.Count(got, tc.want) != 1 {
				t.Fatalf("want one %q:\n%s", tc.want, got)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional external task lease display
func TestTaskLeaseStaysOutOfTheStateCache(t *testing.T) {
	st := &State{TaskLease: &TaskLease{ID: "abc"}, TaskLeaseError: "reason"}
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "abc") || strings.Contains(string(data), "reason") {
		t.Fatalf("task lease persisted: %s", data)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Machine-readable writer lease
func TestWriterLeaseLive(t *testing.T) {
	// Oracle: the requirement. Identity never matters; a foreign session's lease on a live PID is live (#284).
	clearAgentSessionEnv(t)
	now := time.Now()
	alive := os.Getppid()
	for _, tc := range []struct {
		name  string
		lease *WriterLease
		want  bool
	}{
		{"nil", nil, false},
		{"no agent", &WriterLease{PID: alive, LastSeen: now}, false},
		{"foreign live", &WriterLease{Agent: "codex", SessionID: "foreign", PID: alive, LastSeen: now}, true},
		{"exited pid", &WriterLease{Agent: "codex", PID: 999999, LastSeen: now}, false},
		{"past ttl", &WriterLease{Agent: "codex", PID: alive, LastSeen: now.Add(-writerLeaseTTL)}, false},
		{"within ttl", &WriterLease{Agent: "codex", PID: alive, LastSeen: now.Add(-writerLeaseTTL + time.Minute)}, true},
		{"no pid, no last_seen", &WriterLease{Agent: "codex"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := writerLeaseLive(tc.lease); got != tc.want {
				t.Fatalf("writerLeaseLive = %v, want %v", got, tc.want)
			}
		})
	}
}
