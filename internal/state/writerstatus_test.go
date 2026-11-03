package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// spec: openspec/specs/harness/spec.md#Requirement: Machine-readable writer state
func TestClassifyWriterLease(t *testing.T) {
	// Oracle: the D7 table in the design, written out here rather than derived from the code.
	clearAgentSessionEnv(t)
	host, ns := processPlace()
	if host == "" {
		t.Skip("no hostname on this machine")
	}
	now := time.Now()
	old := now.Add(-writerLeaseTTL - time.Minute)
	alive := os.Getppid()
	const dead = 999999
	for _, tc := range []struct {
		name          string
		lease         WriterLease
		state, reason string
	}{
		{"no agent", WriterLease{Host: host, PIDNamespace: ns, PID: alive, LastSeen: now}, "unknown", "lease incomplete"},
		{"no last_seen", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, PID: alive}, "unknown", "lease incomplete"},
		{"no host, within ttl", WriterLease{Agent: "a", PID: alive, LastSeen: now}, "unknown", "no host recorded"},
		{"no host, dead pid within ttl", WriterLease{Agent: "a", PID: dead, LastSeen: now}, "unknown", "no host recorded"},
		{"no host, past ttl", WriterLease{Agent: "a", PID: alive, LastSeen: old}, "stale", "past ttl"},
		{"other host, within ttl, dead here", WriterLease{Agent: "a", Host: host + "x", PIDNamespace: ns, PID: dead, LastSeen: now}, "unknown", "other host"},
		{"other host, within ttl, alive here", WriterLease{Agent: "a", Host: host + "x", PIDNamespace: ns, PID: alive, LastSeen: now}, "unknown", "other host"},
		{"other host, past ttl", WriterLease{Agent: "a", Host: host + "x", PIDNamespace: ns, PID: alive, LastSeen: old}, "stale", "past ttl"},
		{"other namespace, within ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns + "x", PID: alive, LastSeen: now}, "unknown", "other pid namespace"},
		{"other namespace, past ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns + "x", PID: alive, LastSeen: old}, "stale", "past ttl"},
		{"no pid", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, LastSeen: now}, "unknown", "no pid recorded"},
		{"no pid past ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, LastSeen: old}, "unknown", "no pid recorded"},
		{"negative pid past ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, PID: -1, LastSeen: old}, "unknown", "no pid recorded"},
		{"no agent past ttl", WriterLease{Host: host, PIDNamespace: ns, PID: alive, LastSeen: old}, "unknown", "lease incomplete"},
		{"exited pid", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, PID: dead, LastSeen: now}, "stale", "pid exited"},
		{"exited pid past ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, PID: dead, LastSeen: old}, "stale", "pid exited"},
		{"live pid past ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, PID: alive, LastSeen: old}, "unknown", "pid alive past ttl"},
		{"live, foreign session", WriterLease{Agent: "a", SessionID: "foreign", Host: host, PIDNamespace: ns, PID: alive, LastSeen: now}, "live", ""},
		{"just inside ttl", WriterLease{Agent: "a", Host: host, PIDNamespace: ns, PID: alive, LastSeen: now.Add(-writerLeaseTTL + time.Minute)}, "live", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lease := tc.lease
			state, reason := classifyWriterLease(&lease)
			if state != tc.state || reason != tc.reason {
				t.Fatalf("classify = %s/%q, want %s/%q", state, reason, tc.state, tc.reason)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Machine-readable writer state
func TestWriteWriterLeaseRecordsPlaceAndReadsLive(t *testing.T) {
	// A lease this binary writes must classify as live from the same place, or the daemon never sees live.
	clearAgentSessionEnv(t)
	dir := t.TempDir()
	lease, err := WriteWriterLease(dir, "claude", "s")
	if err != nil {
		t.Fatal(err)
	}
	host, ns := processPlace()
	if lease.Host != host || lease.PIDNamespace != ns {
		t.Fatalf("lease place = %q/%q, want %q/%q", lease.Host, lease.PIDNamespace, host, ns)
	}
	if got := ReadWriterStatus(dir); got.State != WriterStateLive {
		t.Fatalf("state = %s (%s), want live", got.State, got.Reason)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ateam", "writer.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadWriterStatus(dir); got.State != WriterStateUnknown || got.Reason != "lease unreadable" {
		t.Fatalf("corrupt lease = %s/%q, want unknown/lease unreadable", got.State, got.Reason)
	}
}
