// Package state manages hybrid repository state across git worktrees, GitHub/Gitea, OpenSpec, and the local .ateam cache.
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

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
	Branch        string    `json:"branch"`
	CheckpointSHA string    `json:"checkpoint_sha"`
	Dirty         bool      `json:"dirty"`
	AutoCompacted bool      `json:"auto_compacted,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	Path    string `json:"path"`
	Head    string `json:"head"`
	Branch  string `json:"branch"`
	Agent   string `json:"agent,omitempty"`
	Subject string `json:"subject,omitempty"`
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
	Mode             DisciplineMode   `json:"mode"`
	Forge            ForgeConfig      `json:"forge"`
	Forges           []ForgeConfig    `json:"forges,omitempty"`
	ForgesConfigured bool             `json:"forges_configured,omitempty"`
	LastHandoff      *Handoff         `json:"last_handoff,omitempty"`
	Writer           *WriterLease     `json:"writer,omitempty"`
	CurrentSessionID string           `json:"-"`
	Git              GitStatus        `json:"git"`
	Cards            []BoardCard      `json:"cards"`
	Shortcuts        []ShortcutItem   `json:"shortcuts,omitempty"`
	Holds            []HoldItem       `json:"holds,omitempty"`
	HoldScanError    string           `json:"hold_scan_error,omitempty"`
	OpenSpecQueue    []OpenSpecChange `json:"openspec_queue,omitempty"`
	OpenSpecSpecs    []OpenSpecSpec   `json:"openspec_specs,omitempty"`
	UpdatedAt        time.Time        `json:"updated_at"`
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
		PID:       os.Getpid(),
		LastSeen:  time.Now().UTC(),
	}
	if os.Getenv("HERDR_ENV") == "1" {
		wl.HerdrPaneID = strings.TrimSpace(os.Getenv("HERDR_PANE_ID"))
		wl.HerdrWorkspaceID = strings.TrimSpace(os.Getenv("HERDR_WORKSPACE_ID"))
	}
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
	}
	st.Writer = nil
	if wl, err := ReadWriterLease(root); err == nil && wl != nil {
		st.Writer = wl
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

// MergeForgeCards merges newly fetched remote cards for provider while preserving local and other-forge cards.
func (st *State) MergeForgeCards(synced []BoardCard, provider ForgeProvider) {
	syncedByID := make(map[string]BoardCard, len(synced))
	for _, c := range synced {
		syncedByID[c.ID] = c
	}
	var merged []BoardCard
	seen := make(map[string]bool, len(st.Cards)+len(synced))
	for _, existing := range st.Cards {
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
		if existing.Forge != string(provider) || strings.HasPrefix(existing.ID, "ATEAM-") {
			merged = append(merged, existing)
			seen[existing.ID] = true
		}
	}
	for _, c := range synced {
		if !seen[c.ID] {
			merged = append(merged, c)
			seen[c.ID] = true
		}
	}
	st.Cards = merged
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
		return nil
	})
}

// refreshForgeConfig seeds automatic detection until the user explicitly edits settings.
func (st *State) refreshForgeConfig(detected ForgeConfig) {
	if st.ForgesConfigured {
		return
	}
	// In automatic mode, the current origin replaces its previously detected target.
	st.Forges = MergeConfiguredForges(ForgeConfig{}, append(st.Forges, detected))
	st.Forge = ForgeConfig{Provider: ForgeLocal}
	for _, cfg := range st.Forges {
		if cfg.Provider == detected.Provider {
			st.Forge = cfg
			break
		}
	}
}

// UpsertForge adds or updates a configured forge provider on the state.
func (st *State) UpsertForge(cfg ForgeConfig) error {
	var err error
	cfg, err = normalizeForgeConfig(cfg)
	if err != nil {
		return err
	}
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
	return nil
}

// RemoveForge removes a configured forge provider from the state.
func (st *State) RemoveForge(provider ForgeProvider) {
	p := ForgeProvider(strings.ToLower(strings.TrimSpace(string(provider))))
	if p != ForgeGitHub && p != ForgeGitea {
		return
	}
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
		if gs.Worktrees[i].Head != "" {
			gs.Worktrees[i].Subject = runGitLine(ctx, root, "log", "-1", "--format=%s", gs.Worktrees[i].Head)
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
	var b strings.Builder
	holdsDisplay := fmt.Sprintf("%d", len(st.Holds))
	if st.HoldScanError != "" {
		holdsDisplay = "unavailable"
	}
	fmt.Fprintf(&b, "[ateam] mode=%s | forge=%s | branch=%s | head=%s | dirty=%v | shortcuts=%d | holds=%s\n",
		st.Mode, st.Forge.Provider, st.Git.Branch, st.Git.HeadSHA, st.Git.Dirty, len(st.Shortcuts), holdsDisplay)
	if st.HoldScanError != "" {
		fmt.Fprintf(&b, "WARNING: Hold/OpenSpec scan failed (%s) — hold status unavailable\n", st.HoldScanError)
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
		fmt.Fprintf(&b, "Writer lease: agent=%s session=%s pid=%d last_seen=%s ago%s\n",
			st.Writer.Agent, st.Writer.SessionID, st.Writer.PID, age, pane)
		curSession := strings.TrimSpace(st.CurrentSessionID)
		if curSession == "" {
			curSession = strings.TrimSpace(os.Getenv("ATEAM_SESSION_ID"))
		}
		leaseSession := strings.TrimSpace(st.Writer.SessionID)
		if time.Since(st.Writer.LastSeen) < 10*time.Minute && leaseSession != "" && leaseSession != curSession {
			fmt.Fprintf(&b, "WARNING: another writer may be active (agent=%s, session=%s, pid=%d, last_seen=%s ago)%s\n",
				st.Writer.Agent, st.Writer.SessionID, st.Writer.PID, age, pane)
		}
	}
	if len(st.OpenSpecQueue) > 0 {
		fmt.Fprintf(&b, "OpenSpec in-flight changes: %d\n", len(st.OpenSpecQueue))
		for _, ch := range st.OpenSpecQueue {
			fmt.Fprintf(&b, "  - %s (%d/%d tasks, %d caveats)\n", ch.Name, ch.CompletedTasks, ch.TotalTasks, len(ch.Caveats))
		}
	}
	if len(st.Holds) > 0 {
		fmt.Fprintf(&b, "Active Hold items:\n")
		for _, h := range st.Holds {
			fmt.Fprintf(&b, "  - %s:%d %s\n", h.File, h.Line, h.Text)
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
	if st.LastHandoff != nil {
		drift := ""
		if st.LastHandoff.CheckpointSHA != "" && st.Git.HeadSHA != "" && st.LastHandoff.CheckpointSHA != st.Git.HeadSHA {
			drift = fmt.Sprintf(" (HEAD drifted from %s -> %s)", st.LastHandoff.CheckpointSHA, st.Git.HeadSHA)
		}
		fmt.Fprintf(&b, "Last Handoff%s:\n%s", drift, FormatFiveFieldPrompt(*st.LastHandoff))
	}
	return strings.TrimSpace(b.String())
}
