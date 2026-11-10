package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SettingsPath is the committed, non-gate home for settings a team chooses (#333). Keys that can weaken a
// merge gate live in .agents/review-policy.json or gate-protected workflow env instead.
const SettingsPath = ".agents/ateam.json"

// maxSettingsBytes caps the file so a stray large commit cannot stall a hook.
const maxSettingsBytes = 64 << 10

// Merge methods accepted by merge_method; each is both a `gh pr merge --<method>` flag and a Gitea `do` value.
const (
	MergeSquash = "squash"
	MergeMerge  = "merge"
	MergeRebase = "rebase"
)

// Settings is the parsed .agents/ateam.json. A zero field means the key is absent and its default applies:
// MergeMethod "" is squash, DefaultBranch "" keeps main and master both default, and CommitStatuses nil is on.
type Settings struct {
	Version        int           `json:"version"`
	MergeMethod    string        `json:"merge_method,omitempty"`
	DefaultBranch  string        `json:"default_branch,omitempty"`
	CommitStatuses *bool         `json:"commit_statuses,omitempty"`
	Forges         []ForgeConfig `json:"forges,omitempty"`
	AteamVersion   string        `json:"ateam_version,omitempty"`
}

// branchNameRe bounds default_branch to plain ref characters, so it can never read as a git option or a range.
var branchNameRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*$`)

// LoadSettings reads root's .agents/ateam.json. A missing file returns zero Settings and ok=false; any other
// read, decode or validation failure returns an error naming the file.
func LoadSettings(root string) (Settings, bool, error) {
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(SettingsPath)))
	if errors.Is(err, os.ErrNotExist) {
		return Settings{}, false, nil
	}
	if err != nil {
		return Settings{}, false, fmt.Errorf("read %s: %w", SettingsPath, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSettingsBytes+1))
	if err != nil {
		return Settings{}, false, fmt.Errorf("read %s: %w", SettingsPath, err)
	}
	s, err := parseSettings(data)
	if err != nil {
		return Settings{}, false, fmt.Errorf("%s: %w", SettingsPath, err)
	}
	return s, true, nil
}

func parseSettings(data []byte) (Settings, error) {
	if len(data) > maxSettingsBytes {
		return Settings{}, fmt.Errorf("file exceeds %d bytes", maxSettingsBytes)
	}
	var s Settings
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Settings{}, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return Settings{}, fmt.Errorf("trailing data after the settings object")
	}
	if s.Version != 1 {
		return Settings{}, fmt.Errorf("version must be 1, got %d", s.Version)
	}
	switch s.MergeMethod {
	case "", MergeSquash, MergeMerge, MergeRebase:
	default:
		return Settings{}, fmt.Errorf("merge_method must be squash, merge or rebase, got %q", s.MergeMethod)
	}
	if b := s.DefaultBranch; b != "" && (!branchNameRe.MatchString(b) || strings.Contains(b, "..") || strings.HasSuffix(b, "/") || strings.HasSuffix(b, ".lock")) {
		return Settings{}, fmt.Errorf("default_branch %q is not a branch name", b)
	}
	if v := s.AteamVersion; v != "" && !IsReleaseTag(v) {
		return Settings{}, fmt.Errorf("ateam_version is %q, not one release such as v0.3.9", v)
	}
	if s.Forges != nil && len(s.Forges) == 0 {
		return Settings{}, fmt.Errorf("forges is an empty list; omit the key to auto-detect the forge")
	}
	seen := map[ForgeProvider]bool{}
	for i, cfg := range s.Forges {
		norm, err := normalizeForgeConfig(cfg)
		if err != nil {
			return Settings{}, fmt.Errorf("forges[%d]: %w", i, err)
		}
		if seen[norm.Provider] {
			return Settings{}, fmt.Errorf("forges[%d]: duplicate provider %s", i, norm.Provider)
		}
		seen[norm.Provider] = true
		s.Forges[i] = norm
	}
	return s, nil
}

// settingsOrDefault is the hook and Refresh read: an invalid file falls back to defaults so it never blocks a
// session, and `ateam doctor --check` is where the error surfaces (D1).
func settingsOrDefault(root string) Settings {
	s, _, err := LoadSettings(root)
	if err != nil {
		return Settings{}
	}
	return s
}

// fallbackDefaultBranches both count as the default branch while default_branch is unset, as before #333
// (OQ1 A); the first labels a checkout with no branch name.
var fallbackDefaultBranches = [...]string{"main", "master"}

// isDefaultBranchName reports whether branch is the default branch: only configured when it is set, else
// main or master.
func isDefaultBranchName(configured, branch string) bool {
	branch = strings.TrimSpace(branch)
	if configured != "" {
		return branch == configured
	}
	return branch == fallbackDefaultBranches[0] || branch == fallbackDefaultBranches[1]
}

// defaultBranchLabel names the default branch where a branch is missing: configured, else main.
func defaultBranchLabel(configured string) string {
	if configured != "" {
		return configured
	}
	return fallbackDefaultBranches[0]
}

// IsDefaultBranch reports whether branch is st's default branch (configured, else main or master).
func (st *State) IsDefaultBranch(branch string) bool { return st.isDefaultBranch(branch) }

// isDefaultBranch is isDefaultBranchName for st's configured branch; a nil st has none configured.
func (st *State) isDefaultBranch(branch string) bool {
	if st == nil {
		return isDefaultBranchName("", branch)
	}
	return isDefaultBranchName(st.Git.DefaultBranch, branch)
}
