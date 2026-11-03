package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	taskLeaseMaxFileBytes  = 64 << 10
	taskLeaseMaxFieldBytes = 256
)

// taskLeaseTokenRe finds the lease a handoff claims in its free-text Addresses (#284).
var taskLeaseTokenRe = regexp.MustCompile(`(?i)\baip-lease:([A-Za-z0-9_-]+)`)

// TaskLease is an external daemon's task lease. ateam reads the file and never writes it, and it decides no
// ownership: the writer lease, the handoff slot, and prune ignore it (#284).
type TaskLease struct {
	ID        string
	Issue     int
	Holder    string
	ExpiresAt time.Time
	Source    string
}

// TaskLeasePath returns ATEAM_TASK_LEASE_FILE (absolute or root-relative) or .ateam/task-lease.json.
func TaskLeasePath(root string) string {
	p := strings.TrimSpace(os.Getenv("ATEAM_TASK_LEASE_FILE"))
	if p == "" {
		return filepath.Join(root, ".ateam", "task-lease.json")
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

// ReadTaskLease returns nil, nil when the file is absent. The file sits in a checkout an agent can edit and its
// values enter agent context, so error text is fixed and never echoes file content.
func ReadTaskLease(root string) (*TaskLease, error) {
	p := TaskLeasePath(root)
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.New("cannot read file")
	}
	// A FIFO at this path would block every hook on open.
	if !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	// shortcut: a FIFO swapped in between Stat and Open can still block, use O_NONBLOCK if this ever happens
	f, err := os.Open(p)
	if err != nil {
		return nil, errors.New("cannot read file")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, taskLeaseMaxFileBytes+1))
	if err != nil {
		return nil, errors.New("cannot read file")
	}
	if len(data) > taskLeaseMaxFileBytes {
		return nil, errors.New("file too large")
	}
	return parseTaskLease(data)
}

func parseTaskLease(data []byte) (*TaskLease, error) {
	var raw struct {
		ID        string `json:"id"`
		Issue     int    `json:"issue"`
		Holder    string `json:"holder"`
		ExpiresAt string `json:"expires_at"`
		Source    string `json:"source"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errors.New("invalid JSON or field type")
	}
	for _, f := range []struct{ name, value string }{
		{"id", raw.ID}, {"holder", raw.Holder}, {"expires_at", raw.ExpiresAt}, {"source", raw.Source},
	} {
		if err := checkTaskLeaseField(f.name, f.value); err != nil {
			return nil, err
		}
	}
	if raw.Issue <= 0 {
		return nil, errors.New("issue must be a positive integer")
	}
	expires, err := time.Parse(time.RFC3339, raw.ExpiresAt)
	if err != nil {
		return nil, errors.New("expires_at is not RFC 3339")
	}
	return &TaskLease{ID: raw.ID, Issue: raw.Issue, Holder: raw.Holder, ExpiresAt: expires, Source: raw.Source}, nil
}

func checkTaskLeaseField(name, value string) error {
	if value == "" {
		return fmt.Errorf("missing %s", name)
	}
	if len(value) > taskLeaseMaxFieldBytes {
		return fmt.Errorf("%s too long", name)
	}
	for _, r := range value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return fmt.Errorf("%s has a control character", name)
		}
	}
	return nil
}

// loadTaskLease fills the per-run task lease fields. A bad file is a warning for pickup, never a load failure.
func (st *State) loadTaskLease(root string) {
	st.TaskLease, st.TaskLeaseError = nil, ""
	lease, err := ReadTaskLease(root)
	if err != nil {
		st.TaskLeaseError = err.Error()
		return
	}
	st.TaskLease = lease
}

func writePickupTaskLease(b *strings.Builder, st *State) {
	if st.TaskLeaseError != "" {
		fmt.Fprintf(b, "WARNING: task lease file unreadable (%s)\n", st.TaskLeaseError)
		return
	}
	lease := st.TaskLease
	if lease == nil {
		return
	}
	suffix := ""
	if !time.Now().Before(lease.ExpiresAt) {
		suffix = " expired"
	}
	fmt.Fprintf(b, "Task lease: issue #%d holder=%s expires=%s (source %s)%s\n",
		lease.Issue, lease.Holder, lease.ExpiresAt.Format(time.RFC3339), lease.Source, suffix)
	cur := st.Handoffs[st.CurrentHandoffKey()]
	if cur == nil {
		return
	}
	if m := taskLeaseTokenRe.FindStringSubmatch(cur.Addresses); m != nil && m[1] != lease.ID {
		fmt.Fprintf(b, "handoff lease %s is not the current task lease %s\n", m[1], lease.ID)
	}
}
