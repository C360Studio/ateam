// Package state manages hybrid repository state across git worktrees, GitHub/Gitea, OpenSpec, and the local .ateam cache.
package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// BuildCommit can be stamped at link time via -ldflags "-X github.com/c360studio/ateam/internal/state.BuildCommit=<sha>".
var BuildCommit string

// BuildRevision returns the short and full VCS commit revision embedded in the binary and whether the build tree was dirty.
func BuildRevision() (shortRev, fullRev string, modified bool) {
	if raw := strings.TrimSpace(BuildCommit); raw != "" {
		fullRev = raw
		shortRev = raw
		if len(shortRev) > 7 {
			shortRev = shortRev[:7]
		}
		return shortRev, fullRev, false
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				fullRev = strings.TrimSpace(s.Value)
			case "vcs.modified":
				modified = strings.TrimSpace(s.Value) == "true"
			}
		}
		if fullRev != "" {
			shortRev = fullRev
			if len(shortRev) > 7 {
				shortRev = shortRev[:7]
			}
			return shortRev, fullRev, modified
		}
	}
	return "unknown", "", false
}

// BuildLabel returns a compact build provenance identifier (e.g. "9f11a42", "9f11a42-dirty", or "unknown").
func BuildLabel() string {
	shortRev, _, modified := BuildRevision()
	if modified && shortRev != "unknown" && shortRev != "dev" {
		return shortRev + "-dirty"
	}
	return shortRev
}

// VersionString formats the human-readable CLI version including build provenance.
func VersionString() string {
	return fmt.Sprintf("ateam v0.1.0 (%s)", BuildLabel())
}

// ReadDiskBinaryBuild inspects ./bin/ateam in root (if present) and returns its short build revision and label.
// With no revision it labels why: "absent" when there is no binary, "unreadable" when --version
// fails or prints no "(build)" — never the running build, which would hide a stale binary (#121).
func ReadDiskBinaryBuild(ctx context.Context, root string) (shortRev, label string) {
	if strings.TrimSpace(root) == "" {
		return "", ""
	}
	binPath := filepath.Join(root, "bin", "ateam")
	if info, err := os.Stat(binPath); err != nil || info.IsDir() {
		return "", "absent"
	}
	// 10s, not 2s: --version exits at once, so this only guards a hung binary.
	// Under a full -race suite a 2s spawn timed out and the running build was
	// reported as the disk build (#116). Not on the <5ms hook path.
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, binPath, "--version").Output()
	if err != nil {
		return "", "unreadable"
	}
	line := strings.TrimSpace(string(out))
	openIdx := strings.LastIndex(line, "(")
	closeIdx := strings.LastIndex(line, ")")
	if openIdx < 0 || closeIdx <= openIdx+1 {
		return "", "unreadable"
	}
	label = strings.TrimSpace(line[openIdx+1 : closeIdx])
	label = strings.TrimSpace(strings.TrimPrefix(label, "build="))
	shortRev = strings.TrimSuffix(label, "-dirty")
	if shortRev == "" {
		return "", "unreadable"
	}
	return shortRev, label
}

// MCPProcess describes a running `ateam mcp` server process recorded in .ateam/mcp-processes.json.
type MCPProcess struct {
	PID         int        `json:"pid"`
	PPID        int        `json:"ppid,omitempty"`
	BuildCommit string     `json:"build_commit"`
	FullCommit  string     `json:"full_commit,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	StartToken  string     `json:"start_token,omitempty"`
}

func mcpProcessesPath(root string) string {
	return filepath.Join(root, ".ateam", "mcp-processes.json")
}

func mcpRegistryRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	if commonDir, ok := GitCommonDir(root); ok {
		if filepath.Base(commonDir) == ".git" {
			if mainRoot := filepath.Dir(commonDir); mainRoot != "" {
				return filepath.Clean(mainRoot)
			}
		}
		return filepath.Clean(commonDir)
	}
	return filepath.Clean(root)
}

func mcpCandidateRoots(root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	seen := make(map[string]bool)
	var roots []string
	addRoot := func(r string) {
		r = strings.TrimSpace(r)
		if r == "" {
			return
		}
		clean := filepath.Clean(r)
		if !seen[clean] {
			seen[clean] = true
			roots = append(roots, clean)
		}
	}
	addRoot(mcpRegistryRoot(root))
	addRoot(root)
	if commonDir, ok := GitCommonDir(root); ok {
		if filepath.Base(commonDir) == ".git" {
			addRoot(filepath.Dir(commonDir))
		}
		wtBase := filepath.Join(commonDir, "worktrees")
		if entries, err := os.ReadDir(wtBase); err == nil {
			for _, ent := range entries {
				if !ent.IsDir() {
					continue
				}
				gitdirFile := filepath.Join(wtBase, ent.Name(), "gitdir")
				raw, err := os.ReadFile(gitdirFile)
				if err != nil {
					continue
				}
				gd := strings.TrimSpace(string(raw))
				if gd == "" {
					continue
				}
				if !filepath.IsAbs(gd) {
					gd = filepath.Join(wtBase, ent.Name(), filepath.FromSlash(gd))
				}
				addRoot(filepath.Dir(filepath.Clean(gd)))
			}
		}
	}
	return roots
}

func readProcessStartAndArgs(pid int) (lstart, args string, ok bool) {
	if pid <= 0 {
		return "", "", false
	}
	cctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(cctx, "ps", "-p", strconv.Itoa(pid), "-o", "lstart=,args=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return "", "", false
	}
	for _, rawLine := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		fields := strings.Fields(strings.TrimSpace(rawLine))
		if len(fields) >= 6 {
			return strings.Join(fields[:5], " "), strings.Join(fields[5:], " "), true
		}
	}
	return "", "", false
}

func isRecordedMCPProcessValid(p MCPProcess) bool {
	if p.PID <= 0 || !IsProcessAlive(p.PID) {
		return false
	}
	if p.PID == os.Getpid() {
		return true
	}
	lstart, args, ok := readProcessStartAndArgs(p.PID)
	if !ok {
		return true
	}
	if strings.TrimSpace(p.StartToken) != "" && lstart != "" && strings.TrimSpace(p.StartToken) != lstart {
		return false
	}
	cmdFields := strings.Fields(args)
	if len(cmdFields) == 0 {
		return false
	}
	base := filepath.Base(cmdFields[0])
	if (base == "ateam" || base == "ateam.exe") && len(cmdFields) >= 2 && cmdFields[1] == "mcp" {
		return true
	}
	if strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".test.exe") {
		return true
	}
	return false
}

func writeMCPProcessesAtomic(root string, procs []MCPProcess) error {
	dir := filepath.Join(root, ".ateam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(procs, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "mcp-processes.json.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, mcpProcessesPath(root)); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// LoadLiveMCPProcesses returns active MCPProcess entries recorded across the repository and its linked worktrees whose OS PIDs are alive and verified.
func LoadLiveMCPProcesses(root string) []MCPProcess {
	roots := mcpCandidateRoots(root)
	if len(roots) == 0 {
		return nil
	}
	seen := make(map[int]bool)
	var live []MCPProcess
	for _, r := range roots {
		data, err := os.ReadFile(mcpProcessesPath(r))
		if err != nil {
			continue
		}
		var all []MCPProcess
		if err := json.Unmarshal(data, &all); err != nil {
			continue
		}
		for _, p := range all {
			if !seen[p.PID] && isRecordedMCPProcessValid(p) {
				seen[p.PID] = true
				live = append(live, p)
			}
		}
	}
	return live
}

// RegisterMCPProcess records the current process in the repository-wide .ateam/mcp-processes.json and returns an unregister cleanup function.
func RegisterMCPProcess(root, buildLabel, fullCommit string) func() {
	regRoot := mcpRegistryRoot(root)
	if regRoot == "" {
		return func() {}
	}
	pid := os.Getpid()
	now := time.Now().UTC()
	lstart, _, _ := readProcessStartAndArgs(pid)
	_ = WithLock(regRoot, func() error {
		live := LoadLiveMCPProcesses(root)
		filtered := make([]MCPProcess, 0, len(live)+1)
		for _, p := range live {
			if p.PID != pid {
				filtered = append(filtered, p)
			}
		}
		filtered = append(filtered, MCPProcess{
			PID:         pid,
			BuildCommit: strings.TrimSpace(buildLabel),
			FullCommit:  strings.TrimSpace(fullCommit),
			StartedAt:   &now,
			StartToken:  lstart,
		})
		return writeMCPProcessesAtomic(regRoot, filtered)
	})
	return func() {
		_ = WithLock(regRoot, func() error {
			live := LoadLiveMCPProcesses(root)
			var kept []MCPProcess
			for _, p := range live {
				if p.PID != pid {
					kept = append(kept, p)
				}
			}
			for _, cand := range mcpCandidateRoots(root) {
				if cand != regRoot {
					_ = os.Remove(mcpProcessesPath(cand))
				}
			}
			if len(kept) == 0 {
				_ = os.Remove(mcpProcessesPath(regRoot))
				return nil
			}
			return writeMCPProcessesAtomic(regRoot, kept)
		})
	}
}

// DisciplineMode represents the active Ponytail decision-ladder level.
type DisciplineMode string

// Supported Ponytail discipline modes.
const (
	ModeLite  DisciplineMode = "lite"
	ModeFull  DisciplineMode = "full"
	ModeUltra DisciplineMode = "ultra"
	ModeOff   DisciplineMode = "off"
)

// NormalizeMode validates and normalizes a discipline mode string.
func NormalizeMode(raw string) (DisciplineMode, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "lite":
		return ModeLite, true
	case "full", "":
		return ModeFull, true
	case "ultra":
		return ModeUltra, true
	case "off", "stop", "normal":
		return ModeOff, true
	default:
		return ModeFull, false
	}
}

// Handoff holds the 5-field SemEngine pickup prompt plus deterministic git metadata.
type Handoff struct {
	Goal          string    `json:"goal"`
	Addresses     string    `json:"addresses"`
	Constraints   string    `json:"constraints"`
	DoneWhen      string    `json:"done_when"`
	ChecksToRun   string    `json:"checks_to_run"`
	Agent         string    `json:"agent,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	Branch        string    `json:"branch"`
	CheckpointSHA string    `json:"checkpoint_sha"`
	Dirty         bool      `json:"dirty"`
	AutoCompacted bool      `json:"auto_compacted,omitempty"`
	Stale         bool      `json:"stale,omitempty"`
	StaleReason   string    `json:"stale_reason,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SameCommitSHA reports whether two commit SHA prefixes refer to the same commit.
func SameCommitSHA(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" || a == "uncommitted" || b == "uncommitted" {
		return a == b
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

var (
	handoffGoalSubjectPRRe = regexp.MustCompile(`(?i)^\s*(?:(?:review(?:\s+and\s+merge)?|merge|land|fix|finish|close)\s+)?(?:\bpr\s*#|\bpull[s]?/)([1-9][0-9]*)\b`)
	handoffPRRefRe         = regexp.MustCompile(`(?i)\b(?:pr\s*#|pull[s]?/)([1-9][0-9]*)\b`)
)

func isPRCard(c BoardCard) bool {
	return c.Type == "pr" || strings.Contains(c.ID, "-PR-") || strings.HasPrefix(c.ID, "PR-")
}

func cardPRNumber(c BoardCard) int {
	if c.Number > 0 {
		return c.Number
	}
	for _, prefix := range []string{"GH-PR-", "GT-PR-", "PR-"} {
		if raw, ok := strings.CutPrefix(c.ID, prefix); ok {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// FormatFiveFieldPrompt formats the canonical 5-field SemEngine pickup block.
func FormatFiveFieldPrompt(h Handoff) string {
	valOrNone := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" {
			return "none"
		}
		return s
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Goal: %s\n", valOrNone(h.Goal))
	fmt.Fprintf(&b, "Addresses: %s\n", valOrNone(h.Addresses))
	fmt.Fprintf(&b, "Constraints: %s\n", valOrNone(h.Constraints))
	fmt.Fprintf(&b, "Done when: %s\n", valOrNone(h.DoneWhen))
	fmt.Fprintf(&b, "Checks to run: %s\n", valOrNone(h.ChecksToRun))
	return b.String()
}

// Worktree describes an active git worktree.
type Worktree struct {
	Path           string `json:"path"`
	Head           string `json:"head"`
	Branch         string `json:"branch"`
	Agent          string `json:"agent,omitempty"`
	Subject        string `json:"subject,omitempty"`
	Dirty          *bool  `json:"dirty,omitempty"`
	AheadCount     int    `json:"ahead_count,omitempty"`
	BehindCount    int    `json:"behind_count,omitempty"`
	MergedIntoMain bool   `json:"merged_into_main,omitempty"`
}

// GitStatus captures the deterministic git and worktree state of a repository.
type GitStatus struct {
	Branch     string     `json:"branch"`
	HeadSHA    string     `json:"head_sha"`
	Dirty      bool       `json:"dirty"`
	DirtyFiles []string   `json:"dirty_files,omitempty"`
	RemoteURL  string     `json:"remote_url,omitempty"`
	UserName   string     `json:"user_name,omitempty"`
	UserEmail  string     `json:"user_email,omitempty"`
	Worktrees  []Worktree `json:"worktrees,omitempty"`
}

// BoardCard represents an issue, PR claim, or OpenSpec item on the PM board.
type BoardCard struct {
	ID         string    `json:"id"`
	Number     int       `json:"number,omitempty"`
	Title      string    `json:"title"`
	Column     string    `json:"column"` // needs-decision, backlog, in-progress, in-review, blocked, done
	Type       string    `json:"type"`   // issue, epic, pr, spec
	Merged     bool      `json:"merged,omitempty"`
	HeadSHA    string    `json:"head_sha,omitempty"`
	Labels     []string  `json:"labels,omitempty"`
	Agent      string    `json:"agent,omitempty"`
	Branch     string    `json:"branch,omitempty"`
	Spec       string    `json:"spec,omitempty"`
	Milestone  string    `json:"milestone,omitempty"`
	HoldReason string    `json:"hold_reason,omitempty"`
	Decision   string    `json:"decision,omitempty"`
	Forge      string    `json:"forge,omitempty"` // github, gitea, local
	URL        string    `json:"url,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WriterLease tracks the most recent active agent writer in .ateam/writer.json.
type WriterLease struct {
	Agent     string    `json:"agent"`
	SessionID string    `json:"session_id,omitempty"`
	PID       int       `json:"pid"`
	LastSeen  time.Time `json:"last_seen"`
	// Herdr IDs are recorded only inside a herdr pane (HERDR_ENV=1); they name a
	// live pane, unlike the short-lived hook PID.
	HerdrPaneID      string `json:"herdr_pane_id,omitempty"`
	HerdrWorkspaceID string `json:"herdr_workspace_id,omitempty"`
}

// State is the cached local state stored in gitignored .ateam/state.json.
type State struct {
	Mode             DisciplineMode              `json:"mode"`
	Forge            ForgeConfig                 `json:"forge"`
	Forges           []ForgeConfig               `json:"forges,omitempty"`
	ForgesConfigured bool                        `json:"forges_configured,omitempty"`
	LastHandoff      *Handoff                    `json:"last_handoff,omitempty"`
	Handoffs         map[string]*Handoff         `json:"handoffs,omitempty"`
	Writer           *WriterLease                `json:"writer,omitempty"`
	CurrentAgent     string                      `json:"-"`
	CurrentSessionID string                      `json:"-"`
	MCPBuild         string                      `json:"-"`
	DiskBuild        string                      `json:"-"`
	MCPStaleTarget   string                      `json:"-"`
	Git              GitStatus                   `json:"git"`
	Cards            []BoardCard                 `json:"cards"`
	BoardSyncedAt    time.Time                   `json:"board_synced_at,omitempty"`
	ForgeSyncedAt    map[ForgeProvider]time.Time `json:"forge_synced_at,omitempty"`
	ForgeSyncTarget  map[ForgeProvider]string    `json:"forge_sync_target,omitempty"`
	BoardSyncError   string                      `json:"board_sync_error,omitempty"`
	Shortcuts        []ShortcutItem              `json:"shortcuts,omitempty"`
	Holds            []HoldItem                  `json:"holds,omitempty"`
	HoldScanError    string                      `json:"hold_scan_error,omitempty"`
	OpenSpecQueue    []OpenSpecChange            `json:"openspec_queue,omitempty"`
	OpenSpecSpecs    []OpenSpecSpec              `json:"openspec_specs,omitempty"`
	UpdatedAt        time.Time                   `json:"updated_at"`
}

func normalizeAgentKey(raw string) string {
	a := strings.ToLower(strings.TrimSpace(raw))
	if a == "antigravity" {
		return "agy"
	}
	return a
}

func resolveAgentFromEnv() string {
	if a := normalizeAgentKey(os.Getenv("ATEAM_AGENT")); a != "" {
		return a
	}
	if strings.TrimSpace(os.Getenv("CLAUDE_SESSION_ID")) != "" || strings.TrimSpace(os.Getenv("CLAUDECODE")) != "" {
		return "claude"
	}
	if strings.TrimSpace(os.Getenv("CODEX_SESSION_ID")) != "" {
		return "codex"
	}
	if strings.TrimSpace(os.Getenv("GEMINI_SESSION_ID")) != "" {
		return "gemini"
	}
	return ""
}

// HandoffSlotKey formats the canonical per-agent, per-branch handoff map key ("<agent>@<branch>").
func HandoffSlotKey(agent, branch string) string {
	a := normalizeAgentKey(agent)
	if a == "" {
		a = inferAgentFromBranch(branch)
	}
	if a == "" {
		a = "agent"
	}
	b := strings.TrimSpace(branch)
	if b == "" {
		b = "main"
	}
	return a + "@" + b
}

func isCallerActiveWriterLease(lease *WriterLease, callerSessionID string) bool {
	if lease == nil || strings.TrimSpace(lease.Agent) == "" {
		return false
	}
	if lease.PID > 0 && !IsProcessAlive(lease.PID) {
		return false
	}
	if !lease.LastSeen.IsZero() && time.Since(lease.LastSeen) >= 10*time.Minute {
		return false
	}
	callerSession := strings.TrimSpace(callerSessionID)
	if callerSession == "" {
		callerSession = ResolveEnvSessionID()
	}
	leaseSession := strings.TrimSpace(lease.SessionID)
	if callerSession != "" {
		if leaseSession == "" || leaseSession != callerSession {
			return false
		}
		return true
	}
	if os.Getenv("HERDR_ENV") == "1" && lease.HerdrPaneID != "" && lease.HerdrPaneID == strings.TrimSpace(os.Getenv("HERDR_PANE_ID")) {
		return true
	}
	if lease.PID > 0 {
		return lease.PID == ResolveClientPID() || lease.PID == os.Getpid()
	}
	return leaseSession == ""
}

// ResolveActiveAgent resolves the active agent name from an explicit flag/parameter, ATEAM_AGENT,
// an active writer lease owned by the caller, the branch prefix (claude/*, codex/*, gemini/*, agy/*), or "agent".
func ResolveActiveAgent(root, explicit, branch string, lease *WriterLease) string {
	return ResolveActiveAgentForSession(root, explicit, "", branch, lease)
}

// ResolveActiveAgentForSession resolves the active agent name using callerSessionID to validate writer-lease ownership.
func ResolveActiveAgentForSession(root, explicit, callerSessionID, branch string, lease *WriterLease) string {
	if a := normalizeAgentKey(explicit); a != "" {
		return a
	}
	if a := resolveAgentFromEnv(); a != "" {
		return a
	}
	if lease == nil && strings.TrimSpace(root) != "" {
		if wl, err := ReadWriterLease(root); err == nil {
			lease = wl
		}
	}
	callerLease := isCallerActiveWriterLease(lease, callerSessionID)
	if callerLease && normalizeAgentKey(lease.Agent) != "agent" {
		return normalizeAgentKey(lease.Agent)
	}
	if strings.TrimSpace(branch) == "" && strings.TrimSpace(root) != "" {
		branch, _ = ReadFastGitHead(root)
	}
	if a := inferAgentFromBranch(branch); a != "" {
		return a
	}
	if callerLease {
		return normalizeAgentKey(lease.Agent)
	}
	return "agent"
}

// EffectiveAgent returns the resolved agent identity for st.
func (st *State) EffectiveAgent() string {
	if st == nil {
		return "agent"
	}
	return ResolveActiveAgentForSession("", st.CurrentAgent, st.CurrentSessionID, st.Git.Branch, st.Writer)
}

func (st *State) migrateLegacyLastHandoff() {
	if st == nil || st.Handoffs != nil || st.LastHandoff == nil {
		return
	}
	if st.LastHandoff.Stale && st.LastHandoff.CheckpointSHA == "" && st.LastHandoff.Branch == "" {
		return
	}
	prev := *st.LastHandoff
	prevBranch := strings.TrimSpace(prev.Branch)
	if prevBranch == "" {
		prevBranch = strings.TrimSpace(st.Git.Branch)
	}
	if prevBranch == "" {
		prevBranch = "main"
	}
	prev.Branch = prevBranch
	prevAgent := normalizeAgentKey(prev.Agent)
	if prevAgent == "" {
		prevAgent = inferAgentFromBranch(prevBranch)
	}
	if prevAgent == "" {
		prevAgent = "agent"
	}
	prev.Agent = prevAgent
	prev.Stale = false
	prev.StaleReason = ""
	slotKey := HandoffSlotKey(prevAgent, prevBranch)
	if len(st.HandoffStaleReasonsFor(slotKey, &prev)) > 0 {
		return
	}
	st.Handoffs = map[string]*Handoff{
		slotKey: &prev,
	}
}

// CurrentHandoffKey returns the "<agent>@<branch>" slot key for the current session on st.
func (st *State) CurrentHandoffKey() string {
	if st == nil {
		return HandoffSlotKey("agent", "main")
	}
	return HandoffSlotKey(st.EffectiveAgent(), st.Git.Branch)
}

// SyncBranchHead updates Git.Branch, Git.HeadSHA, and any matching Worktree.Head entry for branch,
// reporting whether a cached worktree Head SHA changed.
func (st *State) SyncBranchHead(branch, sha string) bool {
	if st == nil {
		return false
	}
	branch = strings.TrimSpace(branch)
	sha = strings.TrimSpace(sha)
	if branch != "" {
		st.Git.Branch = branch
	}
	if sha != "" {
		st.Git.HeadSHA = sha
	}
	return st.syncWorktreeHead(st.Git.Branch, st.Git.HeadSHA)
}

func (st *State) syncWorktreeHead(branch, sha string) bool {
	branch = strings.TrimSpace(branch)
	sha = strings.TrimSpace(sha)
	if st == nil || branch == "" || sha == "" || sha == "uncommitted" {
		return false
	}
	changed := false
	for i := range st.Git.Worktrees {
		if strings.TrimSpace(st.Git.Worktrees[i].Branch) == branch {
			if st.Git.Worktrees[i].Head != sha {
				st.Git.Worktrees[i].Head = sha
				changed = true
			}
		}
	}
	return changed
}

// SetHandoff records h in st.Handoffs under its "<agent>@<branch>" slot and aliases st.LastHandoff to that slot.
func (st *State) SetHandoff(h Handoff) {
	if st == nil {
		return
	}
	if st.Handoffs == nil && st.LastHandoff != nil {
		st.migrateLegacyLastHandoff()
	}
	branch := strings.TrimSpace(h.Branch)
	if branch == "" {
		branch = strings.TrimSpace(st.Git.Branch)
	}
	if branch == "" {
		branch = "main"
	}
	h.Branch = branch
	agent := normalizeAgentKey(h.Agent)
	if agent == "" {
		agent = st.EffectiveAgent()
	}
	h.Agent = agent
	st.syncWorktreeHead(branch, h.CheckpointSHA)
	if st.Handoffs == nil {
		st.Handoffs = make(map[string]*Handoff)
	}
	cp := h
	key := HandoffSlotKey(agent, branch)
	st.Handoffs[key] = &cp
	st.LastHandoff = &cp
}

func (st *State) effectiveSessionID() string {
	if st == nil {
		return ResolveEnvSessionID()
	}
	if s := strings.TrimSpace(st.CurrentSessionID); s != "" {
		return s
	}
	if s := ResolveEnvSessionID(); s != "" {
		return s
	}
	if isCallerActiveWriterLease(st.Writer, "") && strings.TrimSpace(st.Writer.SessionID) != "" {
		return strings.TrimSpace(st.Writer.SessionID)
	}
	return ""
}

func splitHandoffSlotKey(key string) (agent, branch string) {
	if a, b, ok := strings.Cut(strings.TrimSpace(key), "@"); ok {
		return normalizeAgentKey(a), strings.TrimSpace(b)
	}
	return "", strings.TrimSpace(key)
}

func (st *State) headSHAForBranch(branch string) string {
	if st == nil {
		return ""
	}
	branch = strings.TrimSpace(branch)
	if branch == "" || branch == strings.TrimSpace(st.Git.Branch) {
		return strings.TrimSpace(st.Git.HeadSHA)
	}
	for _, wt := range st.Git.Worktrees {
		if strings.TrimSpace(wt.Branch) == branch && strings.TrimSpace(wt.Head) != "" {
			return strings.TrimSpace(wt.Head)
		}
	}
	return ""
}

// HandoffStaleReasonsFor returns human-readable reasons why handoff h in slotKey ("<agent>@<branch>", or "" for current branch) is stale, or nil if fresh.
func (st *State) HandoffStaleReasonsFor(slotKey string, h *Handoff) []string {
	if st == nil || h == nil {
		return nil
	}
	_, slotBranch := splitHandoffSlotKey(slotKey)
	expectedBranch := slotBranch
	if expectedBranch == "" {
		expectedBranch = strings.TrimSpace(st.Git.Branch)
	}
	headSHA := st.headSHAForBranch(expectedBranch)
	var reasons []string
	headDrifted := h.CheckpointSHA != "" && headSHA != "" && !SameCommitSHA(h.CheckpointSHA, headSHA)
	if headDrifted {
		reasons = append(reasons, fmt.Sprintf("HEAD drifted from %s -> %s", h.CheckpointSHA, headSHA))
	}
	branchChanged := h.Branch != "" && expectedBranch != "" && h.Branch != expectedBranch
	if branchChanged {
		reasons = append(reasons, fmt.Sprintf("branch changed from %s -> %s", h.Branch, expectedBranch))
	}
	curSession := st.effectiveSessionID()
	sameSessionAtHead := !headDrifted && !branchChanged && h.CheckpointSHA != "" && headSHA != "" && h.Branch != "" && expectedBranch != "" && curSession != "" && strings.TrimSpace(h.SessionID) == curSession
	if !sameSessionAtHead {
		subjectPRs := make(map[int]bool)
		if m := handoffGoalSubjectPRRe.FindStringSubmatch(h.Goal); len(m) >= 2 {
			if prNum, err := strconv.Atoi(m[1]); err == nil && prNum > 0 {
				subjectPRs[prNum] = true
			}
		}
		if h.Branch != "" && h.Branch != "main" && h.Branch != "master" {
			hasOpenBranchPR := false
			for _, c := range st.Cards {
				if isPRCard(c) && c.Branch == h.Branch && c.Column != "done" {
					hasOpenBranchPR = true
					break
				}
			}
			if !hasOpenBranchPR {
				for _, c := range st.Cards {
					if isPRCard(c) && c.Branch == h.Branch && c.Column == "done" {
						if prNum := cardPRNumber(c); prNum > 0 {
							subjectPRs[prNum] = true
						}
					}
				}
			}
		}
		for _, c := range st.Cards {
			prNum := cardPRNumber(c)
			if isPRCard(c) && prNum > 0 && subjectPRs[prNum] && c.Column == "done" {
				reasons = append(reasons, fmt.Sprintf("PR #%d is closed/merged", prNum))
				delete(subjectPRs, prNum)
			}
		}
	}
	if len(reasons) == 0 && h.Stale {
		if strings.TrimSpace(h.StaleReason) != "" {
			reasons = append(reasons, strings.TrimSpace(h.StaleReason))
		} else {
			reasons = append(reasons, "marked stale")
		}
	}
	return reasons
}

// HandoffStaleReasons returns human-readable reasons why st.LastHandoff is stale, or nil if fresh.
func (st *State) HandoffStaleReasons() []string {
	if st == nil || st.LastHandoff == nil {
		return nil
	}
	return st.HandoffStaleReasonsFor("", st.LastHandoff)
}

// IsHandoffStale reports whether st.LastHandoff has drifted or references completed work.
func (st *State) IsHandoffStale() bool {
	return len(st.HandoffStaleReasons()) > 0
}

// HasNonHeadDriftHandoffStalenessFor reports whether h in slotKey is stale for any reason other than HEAD SHA drift.
func (st *State) HasNonHeadDriftHandoffStalenessFor(slotKey string, h *Handoff) bool {
	if st == nil || h == nil {
		return false
	}
	if h.Stale && h.CheckpointSHA == "" && h.Branch == "" {
		if strings.TrimSpace(h.StaleReason) == "" {
			return true
		}
		for _, part := range strings.Split(h.StaleReason, ";") {
			r := strings.TrimSpace(part)
			if r != "" && !strings.HasPrefix(r, "HEAD drifted from ") {
				return true
			}
		}
	}
	savedStale, savedReason := h.Stale, h.StaleReason
	if h.CheckpointSHA != "" || h.Branch != "" {
		h.Stale = false
		h.StaleReason = ""
	}
	reasons := st.HandoffStaleReasonsFor(slotKey, h)
	h.Stale, h.StaleReason = savedStale, savedReason
	for _, r := range reasons {
		if !strings.HasPrefix(strings.TrimSpace(r), "HEAD drifted from ") {
			return true
		}
	}
	return false
}

// HasNonHeadDriftHandoffStaleness reports whether st.LastHandoff is stale for any reason other than HEAD SHA drift.
func (st *State) HasNonHeadDriftHandoffStaleness() bool {
	if st == nil || st.LastHandoff == nil {
		return false
	}
	return st.HasNonHeadDriftHandoffStalenessFor("", st.LastHandoff)
}

func (st *State) hasExplicitOrInferredAgent() bool {
	if st == nil {
		return false
	}
	if a := normalizeAgentKey(st.CurrentAgent); a != "" && a != "agent" {
		return true
	}
	if resolveAgentFromEnv() != "" {
		return true
	}
	if isCallerActiveWriterLease(st.Writer, st.CurrentSessionID) && normalizeAgentKey(st.Writer.Agent) != "agent" {
		return true
	}
	if inferAgentFromBranch(st.Git.Branch) != "" {
		return true
	}
	return false
}

func (st *State) syncLastHandoffFromSlots() {
	if st == nil || st.Handoffs == nil {
		return
	}
	curKey := st.CurrentHandoffKey()
	if cur, ok := st.Handoffs[curKey]; ok && cur != nil {
		cp := *cur
		st.LastHandoff = &cp
		return
	}
	if !st.hasExplicitOrInferredAgent() && st.LastHandoff != nil {
		lastBranch := strings.TrimSpace(st.LastHandoff.Branch)
		curBranch := strings.TrimSpace(st.Git.Branch)
		if curBranch == "" || lastBranch == "" || lastBranch == curBranch {
			lastKey := HandoffSlotKey(st.LastHandoff.Agent, lastBranch)
			if cur, ok := st.Handoffs[lastKey]; ok && cur != nil {
				cp := *cur
				st.LastHandoff = &cp
				return
			}
		}
	}
	st.LastHandoff = nil
}

func (st *State) refreshHandoffStaleness(prune bool, dropAllStale bool) {
	if st == nil {
		return
	}
	if st.Handoffs == nil && st.LastHandoff != nil {
		lastBr := strings.TrimSpace(st.LastHandoff.Branch)
		curBr := strings.TrimSpace(st.Git.Branch)
		if normalizeAgentKey(st.LastHandoff.Agent) != "" || inferAgentFromBranch(lastBr) != "" || (lastBr != "" && curBr != "" && lastBr != curBr) {
			st.migrateLegacyLastHandoff()
		}
	}
	if st.Handoffs != nil {
		curSession := st.effectiveSessionID()
		for key, h := range st.Handoffs {
			if h == nil {
				delete(st.Handoffs, key)
				continue
			}
			wasExplicitlyStale := h.Stale && h.CheckpointSHA == "" && h.Branch == ""
			prevReason := h.StaleReason
			h.Stale = false
			h.StaleReason = ""
			reasons := st.HandoffStaleReasonsFor(key, h)
			if len(reasons) > 0 {
				h.Stale = true
				h.StaleReason = strings.Join(reasons, "; ")
				keepForSameSessionPreCompact := !dropAllStale &&
					!st.HasNonHeadDriftHandoffStalenessFor(key, h) &&
					strings.TrimSpace(h.SessionID) != "" &&
					(curSession == "" || strings.TrimSpace(h.SessionID) == curSession)
				if prune && !keepForSameSessionPreCompact {
					delete(st.Handoffs, key)
				}
			} else if wasExplicitlyStale {
				h.Stale = true
				h.StaleReason = prevReason
				if prune {
					delete(st.Handoffs, key)
				}
			}
		}
		st.syncLastHandoffFromSlots()
		return
	}
	if st.LastHandoff == nil {
		return
	}
	h := st.LastHandoff
	wasExplicitlyStale := h.Stale && h.CheckpointSHA == "" && h.Branch == ""
	prevReason := h.StaleReason
	h.Stale = false
	h.StaleReason = ""
	reasons := st.HandoffStaleReasons()
	if len(reasons) > 0 {
		h.Stale = true
		h.StaleReason = strings.Join(reasons, "; ")
	} else if wasExplicitlyStale {
		h.Stale = true
		h.StaleReason = prevReason
	}
}

// RefreshHandoffStaleness updates Stale and StaleReason on st.LastHandoff and drops stale entries from st.Handoffs
// while retaining same-session HEAD-only-drifted entries for #50 pre-compact recovery.
func (st *State) RefreshHandoffStaleness() {
	st.refreshHandoffStaleness(true, false)
}

// PruneStaleHandoffs drops all stale entries from st.Handoffs on pickup and syncs st.LastHandoff.
func (st *State) PruneStaleHandoffs() {
	st.refreshHandoffStaleness(true, true)
}

func (st *State) reconcileLiveBranchHeads(root string) bool {
	if st == nil || strings.TrimSpace(root) == "" {
		return false
	}
	updated := false
	if br, sha := ReadFastGitHead(root); br != "" && sha != "" {
		st.SyncBranchHead(br, sha)
		updated = true
	}
	for i := range st.Git.Worktrees {
		wtPath := strings.TrimSpace(st.Git.Worktrees[i].Path)
		if wtPath == "" {
			continue
		}
		if wtBr, wtSHA := ReadFastGitHead(wtPath); wtBr != "" && wtSHA != "" {
			// ReadFastGitHead names a detached checkout "HEAD"; worktree entries store it as "", as git worktree list does.
			if wtBr == "HEAD" {
				wtBr = ""
			}
			st.Git.Worktrees[i].Branch = wtBr
			st.Git.Worktrees[i].Head = wtSHA
		}
	}
	return updated
}

// ReconcilePickup resolves session and agent identity on st, prunes stale entries from st.Handoffs,
// updates the LastHandoff alias, and persists handoff changes to .ateam/state.json.
func ReconcilePickup(root string, st *State, explicitAgent, explicitSessionID string) {
	if st == nil {
		return
	}
	st.CurrentSessionID = ResolveActiveSessionID(root, explicitSessionID)
	st.CurrentAgent = ResolveActiveAgentForSession(root, explicitAgent, st.CurrentSessionID, st.Git.Branch, st.Writer)
	if st.Handoffs == nil && st.LastHandoff != nil && (normalizeAgentKey(st.LastHandoff.Agent) != "" || inferAgentFromBranch(st.LastHandoff.Branch) != "" || (strings.TrimSpace(st.LastHandoff.Branch) != "" && strings.TrimSpace(st.LastHandoff.Branch) != strings.TrimSpace(st.Git.Branch)) || st.hasExplicitOrInferredAgent()) {
		st.migrateLegacyLastHandoff()
	}
	st.PruneStaleHandoffs()
	if strings.TrimSpace(root) == "" {
		return
	}
	if _, err := os.Stat(CacheFilePath(root)); err != nil {
		return
	}
	if updated, err := Update(root, func(cur *State) error {
		if !cur.reconcileLiveBranchHeads(root) && st.Git.HeadSHA != "" && !cur.UpdatedAt.After(st.UpdatedAt) {
			cur.SyncBranchHead(st.Git.Branch, st.Git.HeadSHA)
		}
		cur.CurrentSessionID = st.CurrentSessionID
		cur.CurrentAgent = ResolveActiveAgentForSession(root, explicitAgent, cur.CurrentSessionID, cur.Git.Branch, cur.Writer)
		if cur.Handoffs == nil && cur.LastHandoff != nil && (normalizeAgentKey(cur.LastHandoff.Agent) != "" || inferAgentFromBranch(cur.LastHandoff.Branch) != "" || (strings.TrimSpace(cur.LastHandoff.Branch) != "" && strings.TrimSpace(cur.LastHandoff.Branch) != strings.TrimSpace(cur.Git.Branch)) || cur.hasExplicitOrInferredAgent()) {
			cur.migrateLegacyLastHandoff()
		}
		cur.PruneStaleHandoffs()
		return nil
	}); err == nil && updated != nil {
		st.Git = updated.Git
		st.Handoffs = updated.Handoffs
		st.LastHandoff = updated.LastHandoff
		st.CurrentAgent = updated.CurrentAgent
	}
}

// ResolveClientPID returns a long-lived client process PID that outlives ephemeral hook subprocesses.
func ResolveClientPID() int {
	if raw := strings.TrimSpace(os.Getenv("ATEAM_CLIENT_PID")); raw != "" {
		if pid, err := strconv.Atoi(raw); err == nil && pid > 0 {
			return pid
		}
	}
	if ppid := os.Getppid(); ppid > 1 {
		return ppid
	}
	return os.Getpid()
}

// ResolveEnvSessionID returns the first non-empty session identifier configured in process environment variables.
func ResolveEnvSessionID() string {
	for _, k := range []string{"ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CODEX_SESSION_ID", "GEMINI_SESSION_ID"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// ResolveActiveSessionID resolves a caller's session identity from an explicit value, environment variables,
// or an active writer lease owned by the same client PID.
func ResolveActiveSessionID(root, explicit string) string {
	if s := strings.TrimSpace(explicit); s != "" {
		return s
	}
	if s := ResolveEnvSessionID(); s != "" {
		return s
	}
	if root != "" {
		if wl, err := ReadWriterLease(root); err == nil && isCallerActiveWriterLease(wl, "") && wl.SessionID != "" {
			return strings.TrimSpace(wl.SessionID)
		}
	}
	return ""
}

// IsProcessAlive reports whether pid refers to a currently running OS process.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// CacheFilePath returns the path to .ateam/state.json under root.
func CacheFilePath(root string) string {
	return filepath.Join(root, ".ateam", "state.json")
}

// WriterLeasePath returns the path to .ateam/writer.json under root.
func WriterLeasePath(root string) string {
	return filepath.Join(root, ".ateam", "writer.json")
}

// ReadWriterLease reads .ateam/writer.json if present.
func ReadWriterLease(root string) (*WriterLease, error) {
	data, err := os.ReadFile(WriterLeasePath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var wl WriterLease
	if err := json.Unmarshal(data, &wl); err != nil {
		return nil, err
	}
	return &wl, nil
}

// ClearWriterLease removes .ateam/writer.json when an agent hands off write ownership.
func ClearWriterLease(root string) error {
	err := os.Remove(WriterLeasePath(root))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WriteWriterLease atomically writes the active writer lease to .ateam/writer.json.
func WriteWriterLease(root, agent, sessionID string) (*WriterLease, error) {
	dir := filepath.Join(root, ".ateam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	agent = strings.TrimSpace(agent)
	if agent == "" {
		agent = strings.TrimSpace(os.Getenv("ATEAM_AGENT"))
	}
	if agent == "" {
		br, _ := ReadFastGitHead(root)
		agent = inferAgentFromBranch(br)
	}
	if agent == "" {
		agent = "agent"
	}
	wl := &WriterLease{
		Agent:     agent,
		SessionID: strings.TrimSpace(sessionID),
		PID:       ResolveClientPID(),
		LastSeen:  time.Now().UTC(),
	}
	if os.Getenv("HERDR_ENV") == "1" {
		wl.HerdrPaneID = strings.TrimSpace(os.Getenv("HERDR_PANE_ID"))
		wl.HerdrWorkspaceID = strings.TrimSpace(os.Getenv("HERDR_WORKSPACE_ID"))
	}
	return writeWriterLeaseStruct(dir, root, wl)
}

func mcpCheckedSessionsPath(root string) string {
	return filepath.Join(root, ".ateam", "mcp-checked-sessions.json")
}

// HasMCPCheckedSession reports whether sessionID has already completed its once-per-session MCP staleness check.
func HasMCPCheckedSession(root, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	data, err := os.ReadFile(mcpCheckedSessionsPath(root))
	if err != nil {
		return false
	}
	var sessions []string
	if err := json.Unmarshal(data, &sessions); err != nil {
		return false
	}
	for _, s := range sessions {
		if strings.TrimSpace(s) == sessionID {
			return true
		}
	}
	return false
}

// RecordMCPCheckedSession records that sessionID has completed its once-per-session MCP staleness check, independent of writer lease ownership.
func RecordMCPCheckedSession(root, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	return WithLock(root, func() error {
		dir := filepath.Join(root, ".ateam")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		var existing []string
		if data, err := os.ReadFile(mcpCheckedSessionsPath(root)); err == nil {
			_ = json.Unmarshal(data, &existing)
		}
		var updated []string
		for _, s := range existing {
			s = strings.TrimSpace(s)
			if s != "" && s != sessionID {
				updated = append(updated, s)
			}
		}
		updated = append(updated, sessionID)
		if len(updated) > 128 {
			updated = updated[len(updated)-128:]
		}
		raw, err := json.Marshal(updated)
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp(dir, "mcp-checked-sessions.json.*.tmp")
		if err != nil {
			return err
		}
		tmpPath := tmp.Name()
		if _, err := tmp.Write(append(raw, '\n')); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
			return err
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
		if err := os.Rename(tmpPath, mcpCheckedSessionsPath(root)); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
		return nil
	})
}

func writeWriterLeaseStruct(dir, root string, wl *WriterLease) (*WriterLease, error) {
	data, err := json.MarshalIndent(wl, "", "  ")
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(dir, "writer.json.*.tmp")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	if err := os.Rename(tmpPath, WriterLeasePath(root)); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	return wl, nil
}

// Load reads .ateam/state.json or returns an initialized default State.
func Load(root string) (*State, error) {
	p := CacheFilePath(root)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			st := defaultState()
			if wl, err := ReadWriterLease(root); err == nil && wl != nil {
				st.Writer = wl
			}
			return st, nil
		}
		return nil, fmt.Errorf("read state cache: %w", err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse state cache: %w", err)
	}
	if mode, ok := NormalizeMode(string(st.Mode)); ok {
		st.Mode = mode
	} else {
		st.Mode = ModeFull
	}
	if st.Cards == nil {
		st.Cards = []BoardCard{}
	} else {
		filtered := make([]BoardCard, 0, len(st.Cards))
		for _, c := range st.Cards {
			if !isLegacyDemoCard(c) {
				filtered = append(filtered, c)
			}
		}
		st.Cards = filtered
	}
	st.Writer = nil
	if wl, err := ReadWriterLease(root); err == nil && wl != nil {
		st.Writer = wl
	}
	if st.Handoffs == nil && st.LastHandoff != nil {
		lastBr := strings.TrimSpace(st.LastHandoff.Branch)
		curBr := strings.TrimSpace(st.Git.Branch)
		if normalizeAgentKey(st.LastHandoff.Agent) != "" || inferAgentFromBranch(lastBr) != "" || (lastBr != "" && curBr != "" && lastBr != curBr) {
			st.migrateLegacyLastHandoff()
		}
	}
	return &st, nil
}

// Save writes st atomically to .ateam/state.json using a unique temporary file.
func Save(root string, st *State) error {
	dir := filepath.Join(root, ".ateam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .ateam dir: %w", err)
	}
	st.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmpFile, err := os.CreateTemp(dir, "state.json.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp state: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(append(data, '\n')); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp state: %w", err)
	}
	if err := os.Rename(tmpPath, CacheFilePath(root)); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit state cache: %w", err)
	}
	return nil
}

// WithLock executes fn while holding an exclusive advisory file lock on .ateam/state.lock.
func WithLock(root string, fn func() error) error {
	dir := filepath.Join(root, ".ateam")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .ateam dir: %w", err)
	}
	lockPath := filepath.Join(dir, "state.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open state lock: %w", err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire state lock: %w", err)
	}
	return fn()
}

// Update atomically loads, mutates, and saves .ateam/state.json under an exclusive file lock.
func Update(root string, fn func(st *State) error) (*State, error) {
	var out *State
	err := WithLock(root, func() error {
		st, err := Load(root)
		if err != nil {
			return err
		}
		if err := fn(st); err != nil {
			return err
		}
		if err := Save(root, st); err != nil {
			return err
		}
		out = st
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func defaultState() *State {
	return &State{
		Mode:      ModeFull,
		Cards:     []BoardCard{},
		UpdatedAt: time.Now().UTC(),
	}
}

// UpsertCard inserts or updates a card on the PM board.
func (st *State) UpsertCard(card BoardCard) (BoardCard, error) {
	card.Title = strings.TrimSpace(card.Title)
	if card.Title == "" {
		return BoardCard{}, fmt.Errorf("card title is required")
	}
	if strings.TrimSpace(card.ID) == "" {
		next := len(st.Cards) + 1
		for {
			candidate := fmt.Sprintf("ATEAM-%d", next)
			conflict := false
			for _, c := range st.Cards {
				if c.ID == candidate {
					conflict = true
					break
				}
			}
			if !conflict {
				card.ID = candidate
				break
			}
			next++
		}
	}
	if strings.TrimSpace(card.Column) == "" {
		card.Column = "backlog"
	}
	if strings.TrimSpace(card.Type) == "" {
		card.Type = "issue"
	}
	if strings.TrimSpace(card.Forge) == "" {
		card.Forge = "local"
	}
	card.UpdatedAt = time.Now().UTC()
	for i := range st.Cards {
		if st.Cards[i].ID == card.ID {
			st.Cards[i] = card
			return card, nil
		}
	}
	st.Cards = append(st.Cards, card)
	return card, nil
}

type legacyDemoSpec struct {
	id        string
	number    int
	title     string
	cardType  string
	agent     string
	branch    string
	milestone string
	forge     string
}

var legacyDemoCards = []legacyDemoSpec{
	{
		id:        "ATEAM-1",
		number:    1,
		title:     "Universal CLI lifecycle hooks (Claude, Codex, Gemini) <5ms",
		cardType:  "epic",
		agent:     "gemini",
		branch:    "gemini/universal-hooks",
		milestone: "v0.1 Foundation",
		forge:     "github",
	},
	{
		id:        "ATEAM-2",
		number:    2,
		title:     "2-Tool multiplexed MCP server (ateam_state, ateam_task) <300 tokens",
		cardType:  "issue",
		agent:     "gemini",
		branch:    "gemini/mcp-server",
		milestone: "v0.1 Foundation",
		forge:     "github",
	},
	{
		id:        "ATEAM-3",
		number:    3,
		title:     "OpenSpec spec:queue / spec:check & Dual-Forge (GitHub + Gitea) sync",
		cardType:  "issue",
		agent:     "gemini",
		branch:    "gemini/openspec-gitea",
		milestone: "v0.1 Foundation",
		forge:     "github",
	},
	{
		id:        "ATEAM-4",
		number:    4,
		title:     "Agentic benchmark suite (Coding Discipline + Handoff Continuity tiers)",
		cardType:  "issue",
		agent:     "claude",
		branch:    "claude/bench-suite",
		milestone: "v0.1 Foundation",
		forge:     "github",
	},
	{
		id:        "ATEAM-5",
		number:    5,
		title:     "Cross-repo automatic PR merge-check & dual-forge webhook enforcement",
		cardType:  "issue",
		agent:     "gemini",
		branch:    "gemini/dual-forge-webhooks",
		milestone: "v0.2 Rollout",
		forge:     "gitea",
	},
}

func isLegacyDemoCard(c BoardCard) bool {
	if c.HoldReason != "" || c.Decision != "" || c.URL != "" || c.Spec != "" {
		return false
	}
	for _, d := range legacyDemoCards {
		if c.ID == d.id &&
			c.Number == d.number &&
			c.Title == d.title &&
			c.Type == d.cardType &&
			c.Agent == d.agent &&
			c.Branch == d.branch &&
			c.Milestone == d.milestone &&
			c.Forge == d.forge {
			return true
		}
	}
	return false
}

func forgeTargetKey(cfg ForgeConfig) string {
	p := ForgeProvider(strings.ToLower(strings.TrimSpace(string(cfg.Provider))))
	if p != ForgeGitHub && p != ForgeGitea {
		return ""
	}
	base := strings.TrimRight(strings.ToLower(strings.TrimSpace(cfg.BaseURL)), "/")
	owner := strings.ToLower(strings.TrimSpace(cfg.Owner))
	repo := strings.ToLower(strings.TrimSpace(cfg.Repo))
	if base == "" && owner == "" && repo == "" {
		return ""
	}
	return fmt.Sprintf("%s|%s|%s|%s", p, base, owner, repo)
}

func (st *State) preserveActiveForgeSync() {
	if st == nil || st.BoardSyncedAt.IsZero() {
		return
	}
	p := st.Forge.Provider
	if p != ForgeGitHub && p != ForgeGitea {
		return
	}
	if st.ForgeSyncedAt == nil {
		st.ForgeSyncedAt = make(map[ForgeProvider]time.Time)
	}
	if _, ok := st.ForgeSyncedAt[p]; !ok {
		st.ForgeSyncedAt[p] = st.BoardSyncedAt.UTC()
	}
	if key := forgeTargetKey(st.Forge); key != "" {
		if st.ForgeSyncTarget == nil {
			st.ForgeSyncTarget = make(map[ForgeProvider]string)
		}
		if st.ForgeSyncTarget[p] == "" {
			st.ForgeSyncTarget[p] = key
		}
	}
}

// SyncedAtFor returns the last successful remote sync timestamp for cfg's provider and target.
func (st *State) SyncedAtFor(cfg ForgeConfig) time.Time {
	if st == nil {
		return time.Time{}
	}
	p := cfg.Provider
	if p != ForgeGitHub && p != ForgeGitea {
		return st.BoardSyncedAt
	}
	if len(st.ForgeSyncTarget) > 0 {
		if recorded := st.ForgeSyncTarget[p]; recorded != "" {
			if cur := forgeTargetKey(cfg); cur != "" && cur != recorded {
				return time.Time{}
			}
		}
	}
	if len(st.ForgeSyncedAt) > 0 {
		if ts, ok := st.ForgeSyncedAt[p]; ok {
			return ts
		}
		return time.Time{}
	}
	if st.Forge.Provider == p {
		return st.BoardSyncedAt
	}
	return time.Time{}
}

// MergeForgeCards reconciles newly fetched remote cards for provider into the board,
// pruning legacy demo cards and stale remote cards belonging to provider while preserving
// legitimate local dashboard cards, cards from the other forge, and local hold/decision fields.
func (st *State) MergeForgeCards(synced []BoardCard, provider ForgeProvider) {
	syncedByID := make(map[string]BoardCard, len(synced))
	for _, c := range synced {
		syncedByID[c.ID] = c
	}
	merged := make([]BoardCard, 0, len(st.Cards)+len(synced))
	seen := make(map[string]bool, len(st.Cards)+len(synced))
	for _, existing := range st.Cards {
		if isLegacyDemoCard(existing) {
			continue
		}
		if updated, ok := syncedByID[existing.ID]; ok {
			if updated.HoldReason == "" && existing.HoldReason != "" {
				updated.HoldReason = existing.HoldReason
			}
			if existing.Column == "needs-decision" && updated.Column != "needs-decision" && updated.Decision == "" {
				updated.Decision = fmt.Sprintf("resolved -> %s", updated.Column)
			}
			if updated.Decision == "" && existing.Decision != "" {
				updated.Decision = existing.Decision
			}
			merged = append(merged, updated)
			seen[existing.ID] = true
			continue
		}
		isLocalTask := strings.HasPrefix(existing.ID, "ATEAM-") || existing.Forge == "" || existing.Forge == string(ForgeLocal)
		isOtherForge := provider != "" && existing.Forge != string(provider)
		if isLocalTask || isOtherForge {
			merged = append(merged, existing)
			seen[existing.ID] = true
		}
	}
	for _, updated := range synced {
		if seen[updated.ID] || isLegacyDemoCard(updated) {
			continue
		}
		merged = append(merged, updated)
		seen[updated.ID] = true
	}
	st.Cards = merged
	st.preserveActiveForgeSync()
	now := time.Now().UTC()
	if provider == ForgeGitHub || provider == ForgeGitea {
		if st.ForgeSyncedAt == nil {
			st.ForgeSyncedAt = make(map[ForgeProvider]time.Time)
		}
		st.ForgeSyncedAt[provider] = now
		if key := forgeTargetKey(st.ResolveForge(provider)); key != "" {
			if st.ForgeSyncTarget == nil {
				st.ForgeSyncTarget = make(map[ForgeProvider]string)
			}
			st.ForgeSyncTarget[provider] = key
		}
	}
	if st.Forge.Provider == "" || st.Forge.Provider == ForgeLocal || st.Forge.Provider == provider {
		st.BoardSyncedAt = now
		st.BoardSyncError = ""
	} else {
		st.BoardSyncedAt = st.SyncedAtFor(st.Forge)
	}
	st.ReconcileWorktrees()
	st.refreshHandoffStaleness(false, false)
}

func selectDoneBranchPR(branchPRs []BoardCard, wtHead string, wantMerged bool) *BoardCard {
	wtHead = strings.TrimSpace(wtHead)
	betterPR := func(cand, cur *BoardCard) bool {
		if cur == nil {
			return true
		}
		candNum, curNum := cardPRNumber(*cand), cardPRNumber(*cur)
		if candNum != curNum {
			return candNum > curNum
		}
		if !cand.UpdatedAt.Equal(cur.UpdatedAt) {
			return cand.UpdatedAt.After(cur.UpdatedAt)
		}
		return true
	}
	var headMatched, fallback *BoardCard
	for i := range branchPRs {
		c := &branchPRs[i]
		if c.Column != "done" || c.Merged != wantMerged {
			continue
		}
		if wtHead != "" && c.HeadSHA != "" && SameCommitSHA(wtHead, c.HeadSHA) {
			if betterPR(c, headMatched) {
				headMatched = c
			}
		} else if betterPR(c, fallback) {
			fallback = c
		}
	}
	if headMatched != nil {
		return headMatched
	}
	return fallback
}

// ReconcileWorktrees updates each Worktree.MergedIntoMain using synced PR cards (matching UI's rule) with branch ancestry (ahead == 0) as fallback.
func (st *State) ReconcileWorktrees() {
	if st == nil {
		return
	}
	for i := range st.Git.Worktrees {
		wt := &st.Git.Worktrees[i]
		branch := strings.TrimSpace(wt.Branch)
		if branch == "" || branch == "main" || branch == "master" || branch == "detached" || branch == "HEAD" {
			continue
		}
		var branchPRs []BoardCard
		for _, c := range st.Cards {
			if isPRCard(c) && c.Branch == branch {
				branchPRs = append(branchPRs, c)
			}
		}
		hasOpenPR := false
		for _, c := range branchPRs {
			if c.Column != "done" {
				hasOpenPR = true
				break
			}
		}
		if hasOpenPR {
			wt.MergedIntoMain = false
			continue
		}
		mergedPR := selectDoneBranchPR(branchPRs, wt.Head, true)
		if mergedPR != nil {
			if mergedPR.HeadSHA != "" && wt.Head != "" {
				if SameCommitSHA(wt.Head, mergedPR.HeadSHA) {
					wt.MergedIntoMain = true
				} else if wt.AheadCount > 0 {
					wt.MergedIntoMain = false
				}
			} else if wt.AheadCount > 0 {
				wt.MergedIntoMain = false
			} else if wt.Head != "" {
				wt.MergedIntoMain = true
			}
			continue
		}
		if wt.AheadCount > 0 {
			wt.MergedIntoMain = false
		}
	}
}

// ForgeCacheStaleThreshold is the age beyond which cached forge PR labels carry a cache-age caveat.
const ForgeCacheStaleThreshold = 10 * time.Minute

func cardForgeProvider(c BoardCard) ForgeProvider {
	p := ForgeProvider(strings.ToLower(strings.TrimSpace(c.Forge)))
	if p == ForgeGitHub || p == ForgeGitea {
		return p
	}
	id := strings.ToUpper(strings.TrimSpace(c.ID))
	if strings.HasPrefix(id, "GH-") {
		return ForgeGitHub
	}
	if strings.HasPrefix(id, "GT-") {
		return ForgeGitea
	}
	return ""
}

func (st *State) forgeConfigForProvider(p ForgeProvider) ForgeConfig {
	if st == nil {
		return ForgeConfig{}
	}
	if p != ForgeGitHub && p != ForgeGitea {
		return st.Forge
	}
	cfg := st.ResolveForge(p)
	if cfg.Provider == p {
		return cfg
	}
	return ForgeConfig{Provider: p}
}

func (st *State) staleForgeCacheAgeFor(p ForgeProvider) string {
	if st == nil {
		return ""
	}
	syncedAt := st.SyncedAtFor(st.forgeConfigForProvider(p))
	if syncedAt.IsZero() {
		return ""
	}
	d := time.Since(syncedAt)
	if d < ForgeCacheStaleThreshold {
		return ""
	}
	return formatCompactAge(d)
}

func withForgeCacheAge(status, age string) string {
	if age != "" {
		return fmt.Sprintf("%s (cache %s)", status, age)
	}
	return status
}

func (st *State) worktreePRStatusAndAge(wt Worktree) (string, string) {
	branch := strings.TrimSpace(wt.Branch)
	// "HEAD" is how caches written before #96 recorded a detached checkout.
	if branch == "" || branch == "detached" || branch == "HEAD" {
		return "detached", ""
	}
	if branch == "main" || branch == "master" {
		return "default branch", ""
	}
	var branchPRs []BoardCard
	if st != nil {
		for _, c := range st.Cards {
			if isPRCard(c) && c.Branch == branch {
				branchPRs = append(branchPRs, c)
			}
		}
	}
	for _, c := range branchPRs {
		if c.Column != "done" {
			age := st.staleForgeCacheAgeFor(cardForgeProvider(c))
			if num := cardPRNumber(c); num > 0 {
				return withForgeCacheAge(fmt.Sprintf("open PR #%d", num), age), age
			}
			return withForgeCacheAge("open PR "+c.ID, age), age
		}
	}
	if c := selectDoneBranchPR(branchPRs, wt.Head, true); c != nil {
		age := st.staleForgeCacheAgeFor(cardForgeProvider(*c))
		num := cardPRNumber(*c)
		prRef := c.ID
		if num > 0 {
			prRef = fmt.Sprintf("#%d", num)
		}
		if c.HeadSHA != "" && wt.Head != "" && !SameCommitSHA(wt.Head, c.HeadSHA) {
			if !wt.MergedIntoMain || wt.AheadCount > 0 {
				return withForgeCacheAge(fmt.Sprintf("ahead of merged %s", prRef), age), age
			}
		} else if c.HeadSHA == "" && wt.AheadCount > 0 && !wt.MergedIntoMain {
			return withForgeCacheAge("unmerged commits ahead of main", age), age
		}
		return withForgeCacheAge(fmt.Sprintf("merged via %s", prRef), age), age
	}
	if c := selectDoneBranchPR(branchPRs, wt.Head, false); c != nil {
		age := st.staleForgeCacheAgeFor(cardForgeProvider(*c))
		if num := cardPRNumber(*c); num > 0 {
			return withForgeCacheAge(fmt.Sprintf("closed without merge (#%d)", num), age), age
		}
		return withForgeCacheAge(fmt.Sprintf("closed without merge (%s)", c.ID), age), age
	}
	if wt.MergedIntoMain && wt.AheadCount == 0 {
		// shortcut: a branch at main's head reads as not started, so a fast-forward merge reads "no commits yet" until main moves, and a branch started from an older main still reads merged; upgrade when the branch's fork point is recorded.
		if head := st.defaultBranchHead(); head != "" && wt.Head != "" && SameCommitSHA(wt.Head, head) {
			return "no commits yet", ""
		}
		return "merged (ancestor of main)", ""
	}
	age := st.staleForgeCacheAgeFor("")
	return withForgeCacheAge("no PR", age), age
}

// defaultBranchHead returns the head of the main or master worktree, else of the current checkout when it is on main or master.
func (st *State) defaultBranchHead() string {
	if st == nil {
		return ""
	}
	for _, wt := range st.Git.Worktrees {
		if b := strings.TrimSpace(wt.Branch); b == "main" || b == "master" {
			return wt.Head
		}
	}
	if b := strings.TrimSpace(st.Git.Branch); b == "main" || b == "master" {
		return st.Git.HeadSHA
	}
	return ""
}

// WorktreePRStatus returns a concise PR/merge status for wt (e.g. "merged via #57", "open PR #58", "ahead of merged #57", "closed without merge (#25)", "merged (ancestor of main)", "no commits yet", "default branch", or "no PR"), appending "(cache <age>)" to forge-derived labels when the supplying forge cache is older than ForgeCacheStaleThreshold.
func (st *State) WorktreePRStatus(wt Worktree) string {
	status, _ := st.worktreePRStatusAndAge(wt)
	return status
}

// PopulateMCPProvenance records the running MCP server's build revision alongside the on-disk ./bin/ateam revision and computes any staleness target SHA.
func (st *State) PopulateMCPProvenance(ctx context.Context, root, mcpLabel string) {
	if st == nil {
		return
	}
	mcpLabel = strings.TrimSpace(mcpLabel)
	if mcpLabel == "" {
		mcpLabel = BuildLabel()
	}
	st.MCPBuild = mcpLabel
	diskShort, diskLabel := ReadDiskBinaryBuild(ctx, root)
	if diskLabel != "" {
		st.DiskBuild = diskLabel
	} else {
		st.DiskBuild = mcpLabel
	}
	mcpShort := strings.TrimSuffix(mcpLabel, "-dirty")
	st.MCPStaleTarget = ""
	if diskShort != "" && diskShort != "unknown" && diskShort != "dev" && !SameCommitSHA(mcpShort, diskShort) {
		st.MCPStaleTarget = diskShort
		return
	}
	headSHA := strings.TrimSpace(st.Git.HeadSHA)
	if headSHA != "" && headSHA != "uncommitted" {
		if mcpShort == "" || mcpShort == "unknown" || mcpShort == "dev" {
			if diskShort != "" && diskShort != "unknown" && diskShort != "dev" {
				st.MCPStaleTarget = diskShort
			} else {
				st.MCPStaleTarget = headSHA
			}
			return
		}
		if !SameCommitSHA(mcpShort, headSHA) {
			if diskShort != "" && diskShort != "unknown" && diskShort != "dev" && !SameCommitSHA(diskShort, mcpShort) {
				st.MCPStaleTarget = diskShort
			} else {
				st.MCPStaleTarget = headSHA
			}
		}
	}
}

// BoardSnapshot wraps the board cards with their last remote sync timestamp and any stale-cache fallback notice.
type BoardSnapshot struct {
	Forge          ForgeProvider `json:"forge"`
	SyncedAt       string        `json:"synced_at"`
	BoardSyncedAt  *time.Time    `json:"board_synced_at,omitempty"`
	CachedFallback bool          `json:"cached_fallback,omitempty"`
	SyncError      string        `json:"sync_error,omitempty"`
	Notice         string        `json:"notice,omitempty"`
	Cards          []BoardCard   `json:"cards"`
}

// BoardView returns a BoardSnapshot including sync metadata and fallback status for CLI/MCP board responses.
func (st *State) BoardView() BoardSnapshot {
	cards := st.Cards
	if cards == nil {
		cards = []BoardCard{}
	}
	syncedAt := "never"
	var tsPtr *time.Time
	activeSync := st.SyncedAtFor(st.Forge)
	if !activeSync.IsZero() {
		utc := activeSync.UTC()
		syncedAt = utc.Format(time.RFC3339)
		tsPtr = &utc
	}
	snap := BoardSnapshot{
		Forge:         st.Forge.Provider,
		SyncedAt:      syncedAt,
		BoardSyncedAt: tsPtr,
		Cards:         cards,
	}
	if strings.TrimSpace(st.BoardSyncError) != "" {
		snap.CachedFallback = true
		snap.SyncError = strings.TrimSpace(st.BoardSyncError)
		snap.Notice = fmt.Sprintf("WARNING: forge sync failed (%s); serving cached board (synced=%s)", snap.SyncError, syncedAt)
	}
	return snap
}

// MoveCard moves an existing card by ID to a new column and optional agent assignee.
func (st *State) MoveCard(id, column, agent string) (BoardCard, error) {
	id = strings.TrimSpace(id)
	column = strings.TrimSpace(column)
	if id == "" || column == "" {
		return BoardCard{}, fmt.Errorf("id and column are required")
	}
	for i := range st.Cards {
		if st.Cards[i].ID == id {
			st.Cards[i].Column = column
			if strings.TrimSpace(agent) != "" {
				st.Cards[i].Agent = strings.TrimSpace(agent)
			}
			st.Cards[i].UpdatedAt = time.Now().UTC()
			return st.Cards[i], nil
		}
	}
	return BoardCard{}, fmt.Errorf("card %q not found", id)
}

// Refresh inspects live git worktrees, shortcuts, OpenSpec queue, and optional forge state.
func Refresh(ctx context.Context, root string, syncRemote bool) (*State, error) {
	cached, err := Load(root)
	if err != nil {
		return nil, err
	}
	gs := SnapshotGit(ctx, root)
	detectedForge := DetectForge(ctx, root, gs.RemoteURL)
	cached.refreshForgeConfig(detectedForge)
	forgeCfg := cached.Forge
	shortcuts, err := ScanShortcuts(root)
	if err != nil {
		return nil, err
	}
	holds, err := ScanHolds(root)
	if err != nil {
		return nil, err
	}
	queue, err := ScanOpenSpecQueue(root)
	if err != nil {
		return nil, err
	}
	specs, err := ScanOpenSpecSpecs(root)
	if err != nil {
		return nil, err
	}
	var syncedCards []BoardCard
	if syncRemote {
		cards, err := SyncForgeBoard(ctx, root, forgeCfg)
		if err != nil {
			return nil, err
		}
		syncedCards = cards
	}
	return Update(root, func(st *State) error {
		st.Git = gs
		st.refreshForgeConfig(detectedForge)
		st.Shortcuts = shortcuts
		st.Holds = holds
		st.HoldScanError = ""
		st.OpenSpecQueue = queue
		st.OpenSpecSpecs = specs
		if syncRemote {
			st.MergeForgeCards(syncedCards, forgeCfg.Provider)
		}
		st.ReconcileWorktrees()
		st.refreshHandoffStaleness(false, false)
		return nil
	})
}

func (st *State) hasCardsForProvider(p ForgeProvider) bool {
	if st == nil || (p != ForgeGitHub && p != ForgeGitea) {
		return false
	}
	for _, c := range st.Cards {
		if cardForgeProvider(c) == p {
			return true
		}
	}
	return false
}

// RefreshWithBestEffortSync refreshes local state and attempts a bounded remote forge sync when a remote forge is configured and the board cache is older than maxAge (falling back to cached cards on sync failure).
func RefreshWithBestEffortSync(ctx context.Context, root string, maxAge time.Duration) (*State, error) {
	st, err := Refresh(ctx, root, false)
	if err != nil {
		return nil, err
	}
	if st.Forge.Provider != ForgeGitHub && st.Forge.Provider != ForgeGitea {
		return st, nil
	}
	var toSync []ForgeConfig
	for _, cfg := range MergeConfiguredForges(st.Forge, st.Forges) {
		syncedAt := st.SyncedAtFor(cfg)
		if cfg.Provider != st.Forge.Provider && syncedAt.IsZero() && !st.hasCardsForProvider(cfg.Provider) {
			continue
		}
		if maxAge > 0 && !syncedAt.IsZero() && time.Since(syncedAt) < maxAge {
			continue
		}
		toSync = append(toSync, cfg)
	}
	if len(toSync) == 0 {
		return st, nil
	}
	type syncedForge struct {
		provider ForgeProvider
		cards    []BoardCard
	}
	var (
		syncedResults []syncedForge
		syncErrors    []string
	)
	for _, cfg := range toSync {
		sctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cards, syncErr := SyncForgeBoard(sctx, root, cfg)
		cancel()
		if syncErr != nil {
			syncErrors = append(syncErrors, syncErr.Error())
			continue
		}
		syncedResults = append(syncedResults, syncedForge{provider: cfg.Provider, cards: cards})
	}
	updated, err := Update(root, func(cur *State) error {
		for _, res := range syncedResults {
			cur.MergeForgeCards(res.cards, res.provider)
		}
		cur.BoardSyncError = strings.Join(syncErrors, "; ")
		return nil
	})
	if err != nil {
		st.BoardSyncError = err.Error()
		return st, nil
	}
	updated.CurrentAgent = st.CurrentAgent
	updated.CurrentSessionID = st.CurrentSessionID
	return updated, nil
}

// refreshForgeConfig seeds automatic detection until the user explicitly edits settings.
func (st *State) refreshForgeConfig(detected ForgeConfig) {
	if st.ForgesConfigured {
		return
	}
	st.preserveActiveForgeSync()
	// In automatic mode, the current origin replaces its previously detected target.
	st.Forges = MergeConfiguredForges(ForgeConfig{}, append(st.Forges, detected))
	st.Forge = ForgeConfig{Provider: ForgeLocal}
	for _, cfg := range st.Forges {
		if cfg.Provider == detected.Provider {
			st.Forge = cfg
			break
		}
	}
	st.BoardSyncedAt = st.SyncedAtFor(st.Forge)
}

// UpsertForge adds or updates a configured forge provider on the state.
func (st *State) UpsertForge(cfg ForgeConfig) error {
	var err error
	cfg, err = normalizeForgeConfig(cfg)
	if err != nil {
		return err
	}
	st.preserveActiveForgeSync()
	st.Forges = MergeConfiguredForges(st.Forge, st.Forges)
	st.ForgesConfigured = true
	updated := false
	for i := range st.Forges {
		if st.Forges[i].Provider == cfg.Provider {
			st.Forges[i] = cfg
			updated = true
			break
		}
	}
	if !updated {
		st.Forges = append(st.Forges, cfg)
	}
	if st.Forge.Provider == "" || st.Forge.Provider == ForgeLocal || st.Forge.Provider == cfg.Provider {
		st.Forge = cfg
	}
	st.BoardSyncedAt = st.SyncedAtFor(st.Forge)
	return nil
}

// RemoveForge removes a configured forge provider from the state.
func (st *State) RemoveForge(provider ForgeProvider) {
	p := ForgeProvider(strings.ToLower(strings.TrimSpace(string(provider))))
	if p != ForgeGitHub && p != ForgeGitea {
		return
	}
	st.preserveActiveForgeSync()
	st.ForgesConfigured = true
	var kept []ForgeConfig
	for _, f := range MergeConfiguredForges(st.Forge, st.Forges) {
		if f.Provider != p {
			kept = append(kept, f)
		}
	}
	st.Forges = kept
	if st.Forge.Provider == p {
		if len(st.Forges) > 0 {
			st.Forge = st.Forges[0]
		} else {
			st.Forge = ForgeConfig{Provider: ForgeLocal}
		}
	}
	st.BoardSyncedAt = st.SyncedAtFor(st.Forge)
}

// ResolveForge returns the ForgeConfig for provider from st.Forges, or falls back to st.Forge.
func (st *State) ResolveForge(provider ForgeProvider) ForgeConfig {
	p := ForgeProvider(strings.ToLower(strings.TrimSpace(string(provider))))
	for _, f := range st.Forges {
		if f.Provider == p {
			return f
		}
	}
	if p == "" || p == st.Forge.Provider {
		return st.Forge
	}
	return ForgeConfig{}
}

// SnapshotGit collects current branch, HEAD SHA, dirty files, remote URL, and worktree list.
func SnapshotGit(ctx context.Context, root string) GitStatus {
	var gs GitStatus
	gs.Branch = runGitLine(ctx, root, "branch", "--show-current")
	if gs.Branch == "" {
		gs.Branch = "main"
	}
	gs.HeadSHA = runGitLine(ctx, root, "rev-parse", "--short", "HEAD")
	if gs.HeadSHA == "" {
		gs.HeadSHA = "uncommitted"
	}
	gs.RemoteURL = runGitLine(ctx, root, "remote", "get-url", "origin")
	gs.UserName, gs.UserEmail = resolveGitUser(
		runGitLine(ctx, root, "config", "user.name"),
		runGitLine(ctx, root, "config", "user.email"),
		currentOSUser(),
	)

	porcelain := runGitOutput(ctx, root, "status", "--porcelain")
	if strings.TrimSpace(porcelain) != "" {
		gs.Dirty = true
		for _, line := range strings.Split(strings.TrimSpace(porcelain), "\n") {
			if strings.TrimSpace(line) != "" {
				gs.DirtyFiles = append(gs.DirtyFiles, strings.TrimSpace(line))
			}
		}
	}
	gs.Worktrees = parseWorktrees(runGitOutput(ctx, root, "worktree", "list", "--porcelain"))
	for i := range gs.Worktrees {
		wt := &gs.Worktrees[i]
		if wt.Head != "" {
			wt.Subject = runGitLine(ctx, root, "log", "-1", "--format=%s", wt.Head)
		}
		if wt.Path != "" {
			if info, err := os.Stat(wt.Path); err == nil && info.IsDir() {
				wtDirty := strings.TrimSpace(runGitOutput(ctx, wt.Path, "status", "--porcelain")) != ""
				wt.Dirty = &wtDirty
			}
		}
		if wt.Head != "" && wt.Branch != "main" && wt.Branch != "master" {
			if counts := strings.Fields(runGitLine(ctx, root, "rev-list", "--left-right", "--count", "main..."+wt.Head)); len(counts) == 2 {
				behind, errB := strconv.Atoi(counts[0])
				ahead, errA := strconv.Atoi(counts[1])
				if errB == nil && errA == nil {
					wt.BehindCount = behind
					wt.AheadCount = ahead
					wt.MergedIntoMain = ahead == 0
				}
			}
		}
	}
	return gs
}

func currentOSUser() string {
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	return strings.TrimSpace(os.Getenv("USERNAME"))
}

func resolveGitUser(gitName, gitEmail, osUser string) (name, email string) {
	name = strings.TrimSpace(gitName)
	email = strings.TrimSpace(gitEmail)
	if name == "" && email != "" {
		if prefix, _, ok := strings.Cut(email, "@"); ok && strings.TrimSpace(prefix) != "" {
			name = strings.TrimSpace(prefix)
		}
	}
	if name == "" {
		name = strings.TrimSpace(osUser)
	}
	if name == "" {
		name = "Local User"
	}
	return name, email
}

func parseWorktrees(porcelain string) []Worktree {
	var out []Worktree
	var cur Worktree
	flush := func() {
		if cur.Path != "" {
			cur.Agent = inferAgentFromBranch(cur.Branch)
			out = append(out, cur)
			cur = Worktree{}
		}
	}
	for _, line := range strings.Split(porcelain, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			flush()
			cur.Path = strings.TrimSpace(p)
		} else if h, ok := strings.CutPrefix(line, "HEAD "); ok {
			cur.Head = strings.TrimSpace(h)
			if len(cur.Head) > 8 {
				cur.Head = cur.Head[:8]
			}
		} else if b, ok := strings.CutPrefix(line, "branch "); ok {
			cur.Branch = strings.TrimPrefix(strings.TrimSpace(b), "refs/heads/")
		}
	}
	flush()
	return out
}

func inferAgentFromBranch(branch string) string {
	for _, a := range []string{"claude", "codex", "gemini", "agy"} {
		if strings.HasPrefix(strings.ToLower(branch), a+"/") {
			return a
		}
	}
	return ""
}

func runGitLine(ctx context.Context, root string, args ...string) string {
	return strings.TrimSpace(runGitOutput(ctx, root, args...))
}

func runGitOutput(ctx context.Context, root string, args ...string) string {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// ReadFastGitHead reads .git/HEAD (including linked worktree .git files and commondir refs) directly from disk without spawning git, keeping hook latency <5ms.
func ReadFastGitHead(root string) (branch, shortSHA string) {
	gitDir, commonDir, ok := resolveGitDirs(root)
	if !ok {
		return "", ""
	}
	headBytes, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", ""
	}
	headStr := strings.TrimSpace(string(headBytes))
	if ref, ok := strings.CutPrefix(headStr, "ref:"); ok {
		ref = strings.TrimSpace(ref)
		branch = strings.TrimPrefix(ref, "refs/heads/")
		for _, dir := range []string{gitDir, commonDir} {
			if refBytes, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(ref))); err == nil {
				sha := strings.TrimSpace(string(refBytes))
				if len(sha) >= 7 {
					return branch, sha[:7]
				}
			}
		}
		for _, dir := range []string{gitDir, commonDir} {
			if packed, err := os.ReadFile(filepath.Join(dir, "packed-refs")); err == nil {
				for _, line := range strings.Split(string(packed), "\n") {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") {
						continue
					}
					fields := strings.Fields(line)
					if len(fields) == 2 && fields[1] == ref && len(fields[0]) >= 7 {
						return branch, fields[0][:7]
					}
				}
			}
		}
		return branch, ""
	}
	if len(headStr) >= 7 {
		return "HEAD", headStr[:7]
	}
	return "", ""
}

// GitCommonDir returns the git common dir shared by root and all its linked worktrees,
// reading .git, gitdir:, and commondir files only (no git subprocess).
func GitCommonDir(root string) (string, bool) {
	_, commonDir, ok := resolveGitDirs(root)
	return commonDir, ok
}

func resolveGitDirs(root string) (gitDir, commonDir string, ok bool) {
	dotGit := filepath.Join(root, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return "", "", false
	}
	if info.IsDir() {
		return dotGit, dotGit, true
	}
	data, err := os.ReadFile(dotGit)
	if err != nil {
		return "", "", false
	}
	line := strings.TrimSpace(string(data))
	rawGitDir, found := strings.CutPrefix(line, "gitdir:")
	if !found {
		return "", "", false
	}
	rawGitDir = strings.TrimSpace(rawGitDir)
	if !filepath.IsAbs(rawGitDir) {
		rawGitDir = filepath.Join(root, filepath.FromSlash(rawGitDir))
	}
	gitDir = filepath.Clean(rawGitDir)
	commonDir = gitDir
	if commonBytes, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		cd := strings.TrimSpace(string(commonBytes))
		if cd != "" {
			if !filepath.IsAbs(cd) {
				cd = filepath.Join(gitDir, filepath.FromSlash(cd))
			}
			commonDir = filepath.Clean(cd)
		}
	}
	return gitDir, commonDir, true
}

// BuildPickupSummary creates a concise reconciliation report for SessionStart or pickup.
func BuildPickupSummary(st *State) string {
	st.ReconcileWorktrees()
	if st.Handoffs != nil {
		st.PruneStaleHandoffs()
	}
	var b strings.Builder
	holdsDisplay := fmt.Sprintf("%d", len(st.Holds))
	if st.HoldScanError != "" {
		holdsDisplay = "unavailable"
	}
	syncedDisplay := "never"
	if activeSync := st.SyncedAtFor(st.Forge); !activeSync.IsZero() {
		syncedDisplay = activeSync.UTC().Format(time.RFC3339)
	}
	buildField := fmt.Sprintf("build=%s", BuildLabel())
	if strings.TrimSpace(st.MCPBuild) != "" {
		diskBuild := strings.TrimSpace(st.DiskBuild)
		if diskBuild == "" {
			diskBuild = BuildLabel()
		}
		buildField = fmt.Sprintf("build=%s mcp=%s", diskBuild, strings.TrimSpace(st.MCPBuild))
	}
	fmt.Fprintf(&b, "[ateam] mode=%s | forge=%s | branch=%s | head=%s | dirty=%v | shortcuts=%d | holds=%s | %s | synced=%s\n",
		st.Mode, st.Forge.Provider, st.Git.Branch, st.Git.HeadSHA, st.Git.Dirty, len(st.Shortcuts), holdsDisplay, buildField, syncedDisplay)
	if strings.TrimSpace(st.MCPStaleTarget) != "" {
		fmt.Fprintf(&b, "WARNING: MCP server binary is stale; restart the client to pick up %s\n", strings.TrimSpace(st.MCPStaleTarget))
	}
	if st.HoldScanError != "" {
		fmt.Fprintf(&b, "WARNING: Hold/OpenSpec scan failed (%s) — hold status unavailable\n", st.HoldScanError)
	}
	if strings.TrimSpace(st.BoardSyncError) != "" {
		fmt.Fprintf(&b, "WARNING: Forge board sync failed (serving cached board, synced=%s): %s\n", syncedDisplay, strings.TrimSpace(st.BoardSyncError))
	}
	if st.Writer != nil && !st.Writer.LastSeen.IsZero() {
		age := time.Since(st.Writer.LastSeen).Round(time.Second)
		if age < 0 {
			age = 0
		}
		pane := ""
		if st.Writer.HerdrPaneID != "" {
			pane = fmt.Sprintf(" (herdr pane %s)", st.Writer.HerdrPaneID)
		}
		pidAlive := IsProcessAlive(st.Writer.PID)
		pidSuffix := ""
		if st.Writer.PID > 0 && !pidAlive {
			pidSuffix = " (exited)"
		}
		fmt.Fprintf(&b, "Writer lease: agent=%s session=%s pid=%d%s last_seen=%s ago%s\n",
			st.Writer.Agent, st.Writer.SessionID, st.Writer.PID, pidSuffix, age, pane)
		curSession := strings.TrimSpace(st.CurrentSessionID)
		if curSession == "" {
			curSession = strings.TrimSpace(os.Getenv("ATEAM_SESSION_ID"))
		}
		if curSession == "" {
			curSession = strings.TrimSpace(os.Getenv("CLAUDE_SESSION_ID"))
		}
		leaseSession := strings.TrimSpace(st.Writer.SessionID)
		sameSession := curSession != "" && leaseSession == curSession
		sameClientPID := curSession == "" && st.Writer.PID > 0 && (st.Writer.PID == ResolveClientPID() || st.Writer.PID == os.Getpid())
		sameHerdrPane := os.Getenv("HERDR_ENV") == "1" && st.Writer.HerdrPaneID != "" && st.Writer.HerdrPaneID == strings.TrimSpace(os.Getenv("HERDR_PANE_ID"))
		if time.Since(st.Writer.LastSeen) < 10*time.Minute && pidAlive && leaseSession != "" && !sameSession && !sameClientPID && !sameHerdrPane {
			fmt.Fprintf(&b, "WARNING: another writer may be active (agent=%s, session=%s, pid=%d, last_seen=%s ago)%s\n",
				st.Writer.Agent, st.Writer.SessionID, st.Writer.PID, age, pane)
		}
	}
	writePickupWorktrees(&b, st)
	var inFlightChanges, completedChanges []OpenSpecChange
	for _, ch := range st.OpenSpecQueue {
		if ch.IsCompletedUnarchived() {
			completedChanges = append(completedChanges, ch)
		} else {
			inFlightChanges = append(inFlightChanges, ch)
		}
	}
	if len(inFlightChanges) > 0 {
		fmt.Fprintf(&b, "OpenSpec in-flight changes: %d\n", len(inFlightChanges))
		for _, ch := range inFlightChanges {
			fmt.Fprintf(&b, "  - %s (%d/%d tasks, %d caveats)\n", ch.Name, ch.CompletedTasks, ch.TotalTasks, len(ch.Caveats))
		}
	}
	if len(completedChanges) > 0 {
		fmt.Fprintf(&b, "OpenSpec completed changes (done, not archived): %d\n", len(completedChanges))
		for _, ch := range completedChanges {
			fmt.Fprintf(&b, "  - %s (%d/%d tasks, done, not archived)\n", ch.Name, ch.CompletedTasks, ch.TotalTasks)
		}
	}
	if len(st.Holds) > 0 {
		fmt.Fprintf(&b, "Active Hold items:\n")
		for _, h := range st.Holds {
			fmt.Fprintf(&b, "  - %s:%d %s\n", h.File, h.Line, cleanTaskText(normalizeHoldPrefix(h.Text)))
		}
	}
	var decisions []BoardCard
	for _, c := range st.Cards {
		if strings.TrimSpace(c.Decision) != "" {
			decisions = append(decisions, c)
			if len(decisions) >= 5 {
				break
			}
		}
	}
	if len(decisions) > 0 {
		fmt.Fprintf(&b, "Recent decisions:\n")
		for _, c := range decisions {
			fmt.Fprintf(&b, "  - %s (%s): %s\n", c.ID, c.Title, c.Decision)
		}
	}
	writePickupHandoffs(&b, st)
	return strings.TrimSpace(b.String())
}

func writePickupHandoffs(b *strings.Builder, st *State) {
	curKey := st.CurrentHandoffKey()
	if st.Handoffs != nil {
		if cur := st.Handoffs[curKey]; cur != nil {
			fmt.Fprintf(b, "Last Handoff:\n%s", FormatFiveFieldPrompt(*cur))
		} else {
			fmt.Fprintf(b, "Last Handoff: none for %s\n", curKey)
		}
		var otherKeys []string
		for k, h := range st.Handoffs {
			if k != curKey && h != nil {
				otherKeys = append(otherKeys, k)
			}
		}
		sort.Strings(otherKeys)
		for _, k := range otherKeys {
			fmt.Fprintf(b, "%s\n", st.formatOtherHandoffLine(k, st.Handoffs[k]))
		}
		return
	}
	if st.LastHandoff != nil {
		reasons := st.HandoffStaleReasons()
		if len(reasons) > 0 {
			fmt.Fprintf(b, "Last Handoff (%s): STALE — handoff omitted (%s)\n", reasons[0], strings.Join(reasons, "; "))
		} else {
			legacyBranch := strings.TrimSpace(st.LastHandoff.Branch)
			if legacyBranch == "" {
				legacyBranch = strings.TrimSpace(st.Git.Branch)
			}
			legacyKey := HandoffSlotKey(st.LastHandoff.Agent, legacyBranch)
			if legacyKey == curKey || (normalizeAgentKey(st.LastHandoff.Agent) == "" && !st.hasExplicitOrInferredAgent()) {
				fmt.Fprintf(b, "Last Handoff:\n%s", FormatFiveFieldPrompt(*st.LastHandoff))
			} else {
				fmt.Fprintf(b, "Last Handoff: none for %s\n", curKey)
				fmt.Fprintf(b, "%s\n", st.formatOtherHandoffLine(legacyKey, st.LastHandoff))
			}
		}
		return
	}
	fmt.Fprintf(b, "Last Handoff: none for %s\n", curKey)
}

func formatCompactAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Round(time.Second)/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Round(time.Minute)/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Round(time.Hour)/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func formatHandoffRelativeAge(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	return formatCompactAge(time.Since(t)) + " ago"
}

func (st *State) isPRClosedOnBoard(prNum int) bool {
	if st == nil || prNum <= 0 {
		return false
	}
	for _, c := range st.Cards {
		if isPRCard(c) && cardPRNumber(c) == prNum && c.Column == "done" {
			return true
		}
	}
	return false
}

func (st *State) handoffPRLabel(h *Handoff) string {
	if h == nil {
		return ""
	}
	if st != nil && strings.TrimSpace(h.Branch) != "" && h.Branch != "main" && h.Branch != "master" {
		for _, c := range st.Cards {
			if isPRCard(c) && c.Branch == h.Branch && c.Column != "done" {
				if num := cardPRNumber(c); num > 0 {
					return fmt.Sprintf("PR #%d", num)
				}
			}
		}
	}
	if m := handoffGoalSubjectPRRe.FindStringSubmatch(h.Goal); len(m) >= 2 {
		if prNum, err := strconv.Atoi(m[1]); err == nil && prNum > 0 && !st.isPRClosedOnBoard(prNum) {
			return fmt.Sprintf("PR #%d", prNum)
		}
	}
	for _, s := range []string{h.Addresses, h.Goal} {
		for _, m := range handoffPRRefRe.FindAllStringSubmatch(s, -1) {
			if len(m) >= 2 {
				if prNum, err := strconv.Atoi(m[1]); err == nil && prNum > 0 && !st.isPRClosedOnBoard(prNum) {
					return fmt.Sprintf("PR #%d", prNum)
				}
			}
		}
	}
	return ""
}

func (st *State) formatOtherHandoffLine(key string, h *Handoff) string {
	age := formatHandoffRelativeAge(h.UpdatedAt)
	if pr := st.handoffPRLabel(h); pr != "" {
		return fmt.Sprintf("other handoffs: %s (%s, %s)", key, age, pr)
	}
	return fmt.Sprintf("other handoffs: %s (%s)", key, age)
}

func writePickupWorktrees(b *strings.Builder, st *State) {
	if st == nil || len(st.Git.Worktrees) == 0 {
		return
	}
	fmt.Fprintf(b, "Worktrees: %d\n", len(st.Git.Worktrees))
	statuses := make([]string, len(st.Git.Worktrees))
	var staleAges []string
	seenAges := make(map[string]bool)
	for i, wt := range st.Git.Worktrees {
		status, age := st.worktreePRStatusAndAge(wt)
		statuses[i] = status
		if age != "" && !seenAges[age] {
			seenAges[age] = true
			staleAges = append(staleAges, age)
		}
	}
	if len(staleAges) > 0 {
		fmt.Fprintf(b, "  labels from forge cache synced %s ago; run ateam state pickup\n", strings.Join(staleAges, " / "))
	}
	for i, wt := range st.Git.Worktrees {
		branch := strings.TrimSpace(wt.Branch)
		if branch == "" || branch == "HEAD" {
			branch = "detached"
		}
		dirtyStr := "unknown"
		if wt.Dirty != nil {
			dirtyStr = fmt.Sprintf("%v", *wt.Dirty)
		} else if wt.Branch != "" && wt.Branch == st.Git.Branch && wt.Head != "" && SameCommitSHA(wt.Head, st.Git.HeadSHA) {
			dirtyStr = fmt.Sprintf("%v", st.Git.Dirty)
		}
		fmt.Fprintf(b, "  - %s (head=%s, dirty=%s, ahead=%d, %s)\n", branch, wt.Head, dirtyStr, wt.AheadCount, statuses[i])
	}
}
