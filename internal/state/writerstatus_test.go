package state

import (
	"os"
	"path/filepath"
	"strings"
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

// spec: openspec/specs/harness/spec.md#Requirement: Session start takes the writer lease from the lease itself
func TestCompetingWriter(t *testing.T) {
	t.Setenv("ATEAM_SESSION_ID", "")
	t.Setenv("HERDR_ENV", "1")
	t.Setenv("HERDR_PANE_ID", "p_mine")
	host, pidNS := processPlace()
	now := time.Now().UTC()
	local := WriterLease{Agent: "codex", SessionID: "other", PID: os.Getpid(), LastSeen: now, Host: host, PIDNamespace: pidNS}
	with := func(f func(*WriterLease)) *WriterLease {
		l := local
		f(&l)
		return &l
	}
	for _, tc := range []struct {
		name  string
		lease *WriterLease
		want  bool
	}{
		{"no lease", nil, false},
		{"local live other session", &local, true},
		{"local exited", with(func(l *WriterLease) { l.PID = 999999 }), false},
		{"local no pid", with(func(l *WriterLease) { l.PID = 0 }), false},
		{"local past ttl", with(func(l *WriterLease) { l.LastSeen = now.Add(-time.Hour) }), false},
		{"no session recorded", with(func(l *WriterLease) { l.SessionID = "" }), false},
		{"same session", with(func(l *WriterLease) { l.SessionID = "mine" }), false},
		{"same herdr pane", with(func(l *WriterLease) { l.HerdrPaneID = "p_mine" }), false},
		{"other host, exited pid", with(func(l *WriterLease) { l.Host = "another-host-287"; l.PID = 999999 }), true},
		{"other pid namespace", with(func(l *WriterLease) { l.PIDNamespace = "pid:[1]"; l.PID = 999999 }), true},
		{"no host recorded", with(func(l *WriterLease) { l.Host = ""; l.PID = 999999 }), true},
		{"other host past ttl", with(func(l *WriterLease) { l.Host = "another-host-287"; l.LastSeen = now.Add(-time.Hour) }), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CompetingWriter(&State{Writer: tc.lease, CurrentSessionID: "mine"})
			if (got != nil) != tc.want {
				t.Fatalf("CompetingWriter = %+v, want competing %v", got, tc.want)
			}
			warned := strings.Contains(BuildPickupSummary(&State{Writer: tc.lease, CurrentSessionID: "mine"}), "WARNING: another writer may be active")
			if warned != tc.want {
				t.Fatalf("summary warning = %v, want %v", warned, tc.want)
			}
		})
	}
}
