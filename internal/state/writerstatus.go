package state

import (
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// Writer states printed by `ateam state writer --json`. A daemon treats unknown as occupied (#284).
const (
	WriterStateNone    = "none"
	WriterStateLive    = "live"
	WriterStateStale   = "stale"
	WriterStateUnknown = "unknown"
)

// WriterStatus is advisory: the lease can change hands right after it is read, and nothing enforces it.
type WriterStatus struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
	*WriterLease
}

// processPlace names where this process's PIDs mean anything. Either value is empty where it cannot be read;
// pid_ns is empty off Linux, so same-hostname containers there compare as one place.
func processPlace() (host, pidNamespace string) {
	host, _ = os.Hostname()
	pidNamespace, _ = os.Readlink("/proc/self/ns/pid")
	return host, pidNamespace
}

// ReadWriterStatus never fails: a lease it cannot read is unknown, so only a failed command exits non-zero.
func ReadWriterStatus(root string) WriterStatus {
	lease, err := ReadWriterLease(root)
	if err != nil {
		return WriterStatus{State: WriterStateUnknown, Reason: "lease unreadable"}
	}
	if lease == nil {
		return WriterStatus{State: WriterStateNone}
	}
	state, reason := classifyWriterLease(lease)
	return WriterStatus{State: state, Reason: reason, WriterLease: lease}
}

func classifyWriterLease(lease *WriterLease) (state, reason string) {
	if strings.TrimSpace(lease.Agent) == "" || lease.LastSeen.IsZero() {
		return WriterStateUnknown, "lease incomplete"
	}
	withinTTL := time.Since(lease.LastSeen) < writerLeaseTTL
	if elsewhere := leaseElsewhere(lease); elsewhere != "" {
		if withinTTL {
			return WriterStateUnknown, elsewhere
		}
		return WriterStateStale, "past ttl"
	}
	if lease.PID <= 0 {
		return WriterStateUnknown, "no pid recorded"
	}
	if !IsProcessAlive(lease.PID) {
		return WriterStateStale, "pid exited"
	}
	// last_seen stands still during a turn, so a live PID past the TTL may be a long turn, not an idle session.
	if !withinTTL {
		return WriterStateUnknown, "pid alive past ttl"
	}
	return WriterStateLive, ""
}

// leaseElsewhere names why the lease's PID means nothing here, or returns "" when the lease was written in this
// host and PID namespace.
func leaseElsewhere(lease *WriterLease) string {
	host, pidNamespace := processPlace()
	switch {
	case lease.Host == "":
		return "no host recorded"
	case lease.Host != host:
		return "other host"
	case lease.PIDNamespace != pidNamespace:
		return "other pid namespace"
	}
	return ""
}

// CompetingWriter returns the cached lease when another session may be writing, or nil. Session-start and the
// prompt hooks read it to leave that lease in place, and pickup prints its warning from it, so no rendered text
// decides ownership (#287). A lease from elsewhere counts within the TTL without a PID check (#286).
func CompetingWriter(st *State) *WriterLease {
	w := st.Writer
	if w == nil || w.LastSeen.IsZero() || time.Since(w.LastSeen) >= writerLeaseTTL {
		return nil
	}
	leaseSession := strings.TrimSpace(w.SessionID)
	if leaseSession == "" {
		return nil
	}
	curSession := strings.TrimSpace(st.CurrentSessionID)
	if curSession == "" {
		curSession = ResolveEnvSessionID()
	}
	if curSession != "" && leaseSession == curSession {
		return nil
	}
	if os.Getenv("HERDR_ENV") == "1" && w.HerdrPaneID != "" && w.HerdrPaneID == strings.TrimSpace(os.Getenv("HERDR_PANE_ID")) {
		return nil
	}
	if leaseElsewhere(w) != "" {
		return w
	}
	if !IsProcessAlive(w.PID) {
		return nil
	}
	if curSession == "" && w.PID > 0 && (w.PID == ResolveClientPID() || w.PID == os.Getpid()) {
		return nil
	}
	return w
}

// VersionInfo lets a client tell an older ATeam, whose `version` prints text, from a failed command (#284).
type VersionInfo struct {
	Version      string   `json:"version"`
	Commit       string   `json:"commit"`
	Capabilities []string `json:"capabilities"`
}

// ReadVersionInfo reports this binary's version, commit, and machine-readable capabilities.
func ReadVersionInfo() VersionInfo {
	info, ok := debug.ReadBuildInfo()
	return VersionInfo{
		Version:      releaseVersion(Version, info, ok),
		Commit:       buildLabel(BuildCommit, info, ok),
		Capabilities: []string{"state-writer-json", "task-lease-file"},
	}
}
