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
	Path   string `json:"path"`
	Head   string `json:"head"`
	Branch string `json:"branch"`
	Agent  string `json:"agent,omitempty"`
}

// GitStatus captures the deterministic git and worktree state of a repository.
type GitStatus struct {
	Branch     string     `json:"branch"`
	HeadSHA    string     `json:"head_sha"`
	Dirty      bool       `json:"dirty"`
	DirtyFiles []string   `json:"dirty_files,omitempty"`
	RemoteURL  string     `json:"remote_url,omitempty"`
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
	Milestone  string    `json:"milestone,omitempty"`
	HoldReason string    `json:"hold_reason,omitempty"`
	Forge      string    `json:"forge,omitempty"` // github, gitea, local
	URL        string    `json:"url,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// State is the cached local state stored in gitignored .ateam/state.json.
type State struct {
	Mode          DisciplineMode   `json:"mode"`
	Forge         ForgeConfig      `json:"forge"`
	LastHandoff   *Handoff         `json:"last_handoff,omitempty"`
	Git           GitStatus        `json:"git"`
	Cards         []BoardCard      `json:"cards"`
	Shortcuts     []ShortcutItem   `json:"shortcuts,omitempty"`
	Holds         []HoldItem       `json:"holds,omitempty"`
	OpenSpecQueue []OpenSpecChange `json:"openspec_queue,omitempty"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// CacheFilePath returns the path to .ateam/state.json under root.
func CacheFilePath(root string) string {
	return filepath.Join(root, ".ateam", "state.json")
}

// Load reads .ateam/state.json or returns an initialized default State.
func Load(root string) (*State, error) {
	p := CacheFilePath(root)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			st := defaultState()
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
	if len(st.Cards) == 0 {
		st.Cards = defaultBoardCards()
	}
	return &st, nil
}

// Save writes st atomically to .ateam/state.json.
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
	tmp := filepath.Join(dir, "state.json.tmp")
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	return os.Rename(tmp, CacheFilePath(root))
}

func defaultState() *State {
	return &State{
		Mode:      ModeFull,
		Cards:     defaultBoardCards(),
		UpdatedAt: time.Now().UTC(),
	}
}

func defaultBoardCards() []BoardCard {
	now := time.Now().UTC()
	return []BoardCard{
		{
			ID:        "ATEAM-1",
			Number:    1,
			Title:     "Universal CLI lifecycle hooks (Claude, Codex, Gemini) <5ms",
			Column:    "done",
			Type:      "epic",
			Labels:    []string{"type:epic", "harness"},
			Agent:     "gemini",
			Branch:    "gemini/universal-hooks",
			Milestone: "v0.1 Foundation",
			Forge:     "github",
			UpdatedAt: now,
		},
		{
			ID:        "ATEAM-2",
			Number:    2,
			Title:     "2-Tool multiplexed MCP server (ateam_state, ateam_task) <300 tokens",
			Column:    "done",
			Type:      "issue",
			Labels:    []string{"mcp", "ponytail"},
			Agent:     "gemini",
			Branch:    "gemini/mcp-server",
			Milestone: "v0.1 Foundation",
			Forge:     "github",
			UpdatedAt: now,
		},
		{
			ID:        "ATEAM-3",
			Number:    3,
			Title:     "OpenSpec spec:queue / spec:check & Dual-Forge (GitHub + Gitea) sync",
			Column:    "in-progress",
			Type:      "issue",
			Labels:    []string{"openspec", "gitea", "github"},
			Agent:     "gemini",
			Branch:    "gemini/openspec-gitea",
			Milestone: "v0.1 Foundation",
			Forge:     "github",
			UpdatedAt: now,
		},
		{
			ID:        "ATEAM-4",
			Number:    4,
			Title:     "Agentic benchmark suite (Coding Discipline + Handoff Continuity tiers)",
			Column:    "in-review",
			Type:      "issue",
			Labels:    []string{"benchmark", "ponytail"},
			Agent:     "claude",
			Branch:    "claude/bench-suite",
			Milestone: "v0.1 Foundation",
			Forge:     "github",
			UpdatedAt: now,
		},
		{
			ID:         "ATEAM-5",
			Number:     5,
			Title:      "Cross-repo automatic PR merge-check enforcement rollout",
			Column:     "needs-decision",
			Type:       "issue",
			Labels:     []string{"status:needs-decision", "governance"},
			Milestone:  "v0.2 Rollout",
			HoldReason: "Hold: awaiting owner decision on Gitea branch protection ruleset",
			Forge:      "gitea",
			UpdatedAt:  now,
		},
	}
}

// UpsertCard inserts or updates a card on the PM board.
func (st *State) UpsertCard(card BoardCard) (BoardCard, error) {
	card.Title = strings.TrimSpace(card.Title)
	if card.Title == "" {
		return BoardCard{}, fmt.Errorf("card title is required")
	}
	if strings.TrimSpace(card.ID) == "" {
		card.ID = fmt.Sprintf("ATEAM-%d", len(st.Cards)+1)
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
	st, err := Load(root)
	if err != nil {
		st = defaultState()
	}
	st.Git = SnapshotGit(ctx, root)
	st.Forge = DetectForge(ctx, root, st.Git.RemoteURL)
	if shortcuts, err := ScanShortcuts(root); err == nil {
		st.Shortcuts = shortcuts
	}
	if holds, err := ScanHolds(root); err == nil {
		st.Holds = holds
	}
	if queue, err := ScanOpenSpecQueue(root); err == nil {
		st.OpenSpecQueue = queue
	}
	if syncRemote {
		if cards, err := SyncForgeBoard(ctx, root, st.Forge); err == nil && len(cards) > 0 {
			st.Cards = cards
		}
	}
	_ = Save(root, st)
	return st, nil
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
	return gs
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
	for _, a := range []string{"claude", "codex", "gemini"} {
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

// ReadFastGitHead reads .git/HEAD and loose/packed refs directly from disk without spawning git, keeping hook latency <5ms.
func ReadFastGitHead(root string) (branch, shortSHA string) {
	gitDir := filepath.Join(root, ".git")
	headBytes, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", ""
	}
	headStr := strings.TrimSpace(string(headBytes))
	if ref, ok := strings.CutPrefix(headStr, "ref:"); ok {
		ref = strings.TrimSpace(ref)
		branch = strings.TrimPrefix(ref, "refs/heads/")
		if refBytes, err := os.ReadFile(filepath.Join(gitDir, filepath.FromSlash(ref))); err == nil {
			sha := strings.TrimSpace(string(refBytes))
			if len(sha) >= 7 {
				return branch, sha[:7]
			}
		}
		if packed, err := os.ReadFile(filepath.Join(gitDir, "packed-refs")); err == nil {
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
		return branch, ""
	}
	if len(headStr) >= 7 {
		return "HEAD", headStr[:7]
	}
	return "", ""
}

// BuildPickupSummary creates a concise reconciliation report for SessionStart or pickup.
func BuildPickupSummary(st *State) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[ateam] mode=%s | forge=%s | branch=%s | head=%s | dirty=%v | shortcuts=%d | holds=%d\n",
		st.Mode, st.Forge.Provider, st.Git.Branch, st.Git.HeadSHA, st.Git.Dirty, len(st.Shortcuts), len(st.Holds))
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
	if st.LastHandoff != nil {
		drift := ""
		if st.LastHandoff.CheckpointSHA != "" && st.Git.HeadSHA != "" && st.LastHandoff.CheckpointSHA != st.Git.HeadSHA {
			drift = fmt.Sprintf(" (HEAD drifted from %s -> %s)", st.LastHandoff.CheckpointSHA, st.Git.HeadSHA)
		}
		fmt.Fprintf(&b, "Last Handoff%s:\n%s", drift, FormatFiveFieldPrompt(*st.LastHandoff))
	}
	return strings.TrimSpace(b.String())
}
