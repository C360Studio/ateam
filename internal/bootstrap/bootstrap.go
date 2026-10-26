// Package bootstrap implements ateam init, ateam doctor, and multi-agent setup parity verification.
package bootstrap

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

const (
	maxAgentsMDBytes = 24000
	maxClaudeMDLines = 10
	maxSkillLines    = 25
)

var (
	canonicalSkills = []string{"pickup", "handoff", "preflight", "ponytail", "review-pr"}
	requiredHooks   = []string{"session-start", "before-prompt", "pre-compact", "turn-end"}
)

// Options configures InitRepo.
type Options struct {
	Force          bool
	BootstrapForge bool
	ForgeOverride  string // "", "github", "gitea"

	now func() time.Time // backup timestamp clock; nil means time.Now
}

// InitResult lists the files InitRepo wrote and the backups it made first.
type InitResult struct {
	Written  []string
	BackedUp []string
}

// Adopters write these by hand, so --force keeps a copy before replacing them.
func isHarnessDocument(rel string) bool {
	switch rel {
	case "AGENTS.md", "CLAUDE.md", ".agents/protocol.md", ".agents/rules-index.md":
		return true
	default:
		return false
	}
}

// InitRepo idempotently scaffolds universal skills, hooks, MCP configs, and OpenSpec into root.
func InitRepo(ctx context.Context, root string, opts Options) (InitResult, error) {
	res := InitResult{BackedUp: []string{}}
	now := opts.now
	if now == nil {
		now = time.Now
	}
	stamp := now().Format("20060102-150405")
	files := defaultTemplateFiles()
	if opts.Force {
		// Back up every harness document before writing any file, so nothing is
		// truncated unless every backup succeeded.
		for rel, content := range files {
			if !isHarnessDocument(rel) {
				continue
			}
			backup, err := backupHarnessDocument(filepath.Join(root, filepath.FromSlash(rel)), rel, content, stamp)
			if backup != "" {
				res.BackedUp = append(res.BackedUp, backup)
			}
			if err != nil {
				sort.Strings(res.BackedUp)
				return res, err
			}
		}
		sort.Strings(res.BackedUp)
	}
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		preserve := !opts.Force || isClientConfig(rel)
		if preserve {
			if _, err := os.Lstat(abs); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return res, fmt.Errorf("inspect %s: %w", rel, err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return res, fmt.Errorf("mkdir %s: %w", rel, err)
		}
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if preserve {
			flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
		}
		f, err := os.OpenFile(abs, flags, 0o644)
		if err != nil {
			return res, fmt.Errorf("create %s: %w", rel, err)
		}
		_, writeErr := f.WriteString(content)
		closeErr := f.Close()
		if writeErr != nil {
			return res, fmt.Errorf("write %s: %w", rel, writeErr)
		}
		if closeErr != nil {
			return res, fmt.Errorf("close %s: %w", rel, closeErr)
		}
		res.Written = append(res.Written, rel)
	}

	if err := ensureGitignore(root); err != nil {
		return res, err
	}

	if opts.BootstrapForge {
		gs := state.SnapshotGit(ctx, root)
		cfg := state.DetectForge(ctx, root, gs.RemoteURL)
		if opts.ForgeOverride == "github" {
			cfg.Provider = state.ForgeGitHub
		} else if opts.ForgeOverride == "gitea" {
			cfg.Provider = state.ForgeGitea
		}
		if _, err := state.BootstrapForgeLabels(ctx, root, cfg); err != nil {
			return res, err
		}
	}
	return res, nil
}

// backupHarnessDocument copies abs to <rel>.ateam-backup-<stamp> when it exists
// and differs from the template, returning the backup's relative path. The copy
// and its directory entry are synced before the caller truncates abs. O_EXCL
// refuses to clobber an earlier backup: a collision fails init rather than
// picking another name. A partly written backup is removed; a complete backup
// whose directory sync fails is still returned with the error.
func backupHarnessDocument(abs, rel, template, stamp string) (string, error) {
	existing, err := os.ReadFile(abs)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s for backup: %w", rel, err)
	}
	if string(existing) == template {
		return "", nil
	}
	backupRel := rel + ".ateam-backup-" + stamp
	backupAbs := abs + ".ateam-backup-" + stamp
	f, err := os.OpenFile(backupAbs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("back up %s: %w", rel, err)
	}
	_, writeErr := f.Write(existing)
	syncErr := f.Sync()
	closeErr := f.Close()
	var copyErr error
	switch {
	case writeErr != nil:
		copyErr = fmt.Errorf("write backup %s: %w", backupRel, writeErr)
	case syncErr != nil:
		copyErr = fmt.Errorf("sync backup %s: %w", backupRel, syncErr)
	case closeErr != nil:
		copyErr = fmt.Errorf("close backup %s: %w", backupRel, closeErr)
	}
	if copyErr != nil {
		if err := os.Remove(backupAbs); err != nil {
			return "", fmt.Errorf("%w; remove partial backup: %w", copyErr, err)
		}
		return "", copyErr
	}
	if err := syncDir(filepath.Dir(backupAbs)); err != nil {
		return backupRel, fmt.Errorf("sync directory of backup %s: %w", backupRel, err)
	}
	return backupRel, nil
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	syncErr := d.Sync()
	closeErr := d.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

func ensureGitignore(root string) error {
	p := filepath.Join(root, ".gitignore")
	existing, _ := os.ReadFile(p)
	text := string(existing)
	var add []string
	for _, entry := range []string{".ateam/", ".worktrees/", ".enjoy-logs/", "benchmarks/runs/"} {
		if !strings.Contains(text, entry) {
			add = append(add, entry)
		}
	}
	if len(add) == 0 {
		return nil
	}
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += strings.Join(add, "\n") + "\n"
	return os.WriteFile(p, []byte(text), 0o644)
}

// CheckParity verifies all multi-agent setup, skill adapter, hook, and rule-index invariants.
func CheckParity(root string) []string {
	var out []string
	out = append(out, checkRequiredFiles(root)...)
	out = append(out, checkRulesAndSizeParity(root)...)
	out = append(out, checkSkillAdapters(root)...)
	out = append(out, checkHookConfigs(root)...)
	out = append(out, checkMCPConfigs(root)...)
	out = append(out, checkReviewSetup(root)...)
	return out
}

func checkRequiredFiles(root string) []string {
	var out []string
	required := []string{
		"AGENTS.md",
		"CLAUDE.md",
		".agents/README.md",
		".agents/protocol.md",
		".agents/rules-index.md",
		"openspec/config.yaml",
	}
	for _, rel := range required {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || info.IsDir() || info.Size() == 0 {
			out = append(out, fmt.Sprintf("%s: missing or empty required file", rel))
		}
	}
	if protoData, err := os.ReadFile(filepath.Join(root, ".agents", "protocol.md")); err == nil {
		proto := string(protoData)
		for _, m := range protocolMarkers {
			if !strings.Contains(proto, m.marker) {
				out = append(out, fmt.Sprintf(".agents/protocol.md: missing %q (%s)", m.marker, m.where))
			}
		}
	}
	if readmeData, err := os.ReadFile(filepath.Join(root, "README.md")); err == nil {
		readme := string(readmeData)
		if strings.Contains(readme, "ateam hook") {
			for _, ev := range requiredHooks {
				if !strings.Contains(readme, ev) {
					out = append(out, fmt.Sprintf("README.md: mentions ateam hook but is missing canonical event %q", ev))
				}
			}
		}
	}
	return out
}

// protocolMarkers are what the harness depends on in .agents/protocol.md; headings are the adopter's own.
var protocolMarkers = []struct{ marker, where string }{
	{"Deliverable", "the Clarify step of \"## 2. Work Lifecycle & Definition of Done\" in the shipped protocol template (internal/bootstrap/templates.go)"},
	{"non-goals", "the Clarify step of \"## 2. Work Lifecycle & Definition of Done\" in the shipped protocol template (internal/bootstrap/templates.go)"},
	{"Definition of Done", "the Clarify step of \"## 2. Work Lifecycle & Definition of Done\" in the shipped protocol template (internal/bootstrap/templates.go)"},
	{"Stop boundary", "the Clarify step of \"## 2. Work Lifecycle & Definition of Done\" in the shipped protocol template (internal/bootstrap/templates.go)"},
	{"ateam state handoff", "\"## 1. Hybrid State Architecture\" (session handoff) and the Close step of \"## 2\" in the shipped protocol template"},
	{"openspec/changes/archive/", "the Close step of \"## 2. Work Lifecycle & Definition of Done\" in the shipped protocol template"},
}

type sectionCount struct {
	title string
	rows  int
}

func checkRulesAndSizeParity(root string) []string {
	var out []string
	if claudeData, err := os.ReadFile(filepath.Join(root, "CLAUDE.md")); err == nil {
		claudeText := string(claudeData)
		if !strings.Contains(claudeText, "@AGENTS.md") {
			out = append(out, "CLAUDE.md: missing @AGENTS.md import")
		}
		lines := strings.Count(strings.TrimRight(claudeText, "\n"), "\n") + 1
		if lines > maxClaudeMDLines {
			out = append(out, fmt.Sprintf("CLAUDE.md: %d lines exceeds max %d lines", lines, maxClaudeMDLines))
		}
		if strings.Contains(claudeText, "\n## ") || strings.Contains(claudeText, "\n|") {
			out = append(out, "CLAUDE.md: must not carry its own section heading or rule table")
		}
	}

	agentsData, errA := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	indexData, errI := os.ReadFile(filepath.Join(root, ".agents", "rules-index.md"))
	if errA != nil || errI != nil {
		return out
	}
	if len(agentsData) >= maxAgentsMDBytes {
		out = append(out, fmt.Sprintf("AGENTS.md: %d bytes is not strictly under %d bytes", len(agentsData), maxAgentsMDBytes))
	}
	agents, agentsMixed := parseRuleSectionCounts(string(agentsData))
	index, indexMixed := parseRuleSectionCounts(string(indexData))
	if agentsMixed || indexMixed {
		for _, f := range []struct {
			name  string
			mixed bool
		}{{"AGENTS.md", agentsMixed}, {".agents/rules-index.md", indexMixed}} {
			if f.mixed {
				out = append(out, f.name+": both ## N. and ### N. headings own table rows, so the rule sections are ambiguous; number only the rule section headings")
			}
		}
		return out
	}
	if len(agents) == 0 && len(index) == 0 {
		return append(out, "AGENTS.md and .agents/rules-index.md have no numbered rule sections (## N. or ### N. headings)")
	}
	for _, f := range []struct {
		name     string
		sections []sectionCount
	}{{"AGENTS.md", agents}, {".agents/rules-index.md", index}} {
		for _, s := range f.sections {
			if s.rows == 0 {
				out = append(out, fmt.Sprintf("%s: rule section %q has no table rows", f.name, s.title))
			}
		}
	}
	for i := 0; i < len(agents) || i < len(index); i++ {
		switch {
		case i >= len(index):
			out = append(out, fmt.Sprintf("rule section %q is in AGENTS.md but missing from .agents/rules-index.md", agents[i].title))
		case i >= len(agents):
			out = append(out, fmt.Sprintf("rule section %q is in .agents/rules-index.md but missing from AGENTS.md", index[i].title))
		case agents[i].title != index[i].title:
			out = append(out, fmt.Sprintf("rule section %d: AGENTS.md = %q, .agents/rules-index.md = %q", i+1, agents[i].title, index[i].title))
		case agents[i].rows != index[i].rows:
			out = append(out, fmt.Sprintf("AGENTS.md and .agents/rules-index.md disagree on row count in %q: AGENTS.md has %d, .agents/rules-index.md has %d",
				agents[i].title, agents[i].rows, index[i].rows))
		}
	}
	return out
}

// parseRuleSectionCounts lists the numbered "## N. title" or "### N. title" sections and their table rows.
// Any other level-2 or level-3 heading ends the current section. Only one heading level is returned:
// the one whose numbered sections own table rows (### when neither does), so numbered non-rule headings are
// ignored. When numbered sections at both levels own rows, mixed is true: picking one level could discard
// the real rules and certify a mismatch.
func parseRuleSectionCounts(text string) (sections []sectionCount, mixed bool) {
	type section struct {
		sectionCount
		level int
	}
	var found []section
	cur := -1
	lastRow := false
	for _, line := range strings.Split(text, "\n") {
		// shortcut: a ## line inside a fenced code block ends a section, when an adopter reports it
		level := 3
		title, ok := strings.CutPrefix(line, "### ")
		if !ok {
			level = 2
			title, ok = strings.CutPrefix(line, "## ")
		}
		if ok {
			lastRow = false
			if isNumberedTitle(title) {
				found = append(found, section{sectionCount{title: strings.TrimSpace(title)}, level})
				cur = len(found) - 1
			} else {
				cur = -1
			}
			continue
		}
		row := strings.TrimSpace(line)
		if cur < 0 || !strings.HasPrefix(row, "|") {
			lastRow = false
			continue
		}
		if isTableSeparator(row) {
			if lastRow {
				found[cur].rows-- // the row above a separator is the header
			}
			lastRow = false
			continue
		}
		found[cur].rows++
		lastRow = true
	}
	rows := map[int]int{}
	has := map[int]bool{}
	for _, s := range found {
		rows[s.level] += s.rows
		has[s.level] = true
	}
	if rows[2] > 0 && rows[3] > 0 {
		return nil, true
	}
	best := 3
	if rows[2] > 0 || !has[3] {
		best = 2
	}
	for _, s := range found {
		if s.level == best {
			sections = append(sections, s.sectionCount)
		}
	}
	return sections, false
}

// isTableSeparator matches a trimmed Markdown table separator row: ^\|[\s:|-]+\|$ with at least one -,
// so an all-blank row such as "| | |" counts as a row.
func isTableSeparator(row string) bool {
	return len(row) >= 3 && row[0] == '|' && row[len(row)-1] == '|' && strings.Contains(row, "-") && strings.Trim(row, " \t:|-") == ""
}

func isNumberedTitle(title string) bool {
	i := 0
	for i < len(title) && title[i] >= '0' && title[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(title) && title[i] == '.'
}

var requiredSkillMarkers = map[string][]string{
	"handoff":   {"## 1.", "ateam state handoff", "Goal:"},
	"pickup":    {"## 1.", "ateam state pickup", "Verify Write Ownership"},
	"ponytail":  {"## 1.", "6-Step Decision Ladder", "// shortcut:"},
	"preflight": {"## 1.", "task verify", "Record Evidence"},
	"review-pr": {"## 1.", "reviewed-by:", "Reviewed at"},
}

// skillLinkRe matches a link into .agents/skills/: group 1 is a leading ./ or ../ chain, group 2 the skill
// (never ending in "."), group 3 an optional path inside it.
var skillLinkRe = regexp.MustCompile(`((?:\.\.?/)*)\.agents/skills/([A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*)(/[A-Za-z0-9._/-]*)?`)

// adapterTarget reports every .agents/skills/ link in the adapter at rel that does not name a regular file,
// and returns the skill its first link names, or "" when it has none. A link with a ./ or ../ prefix resolves
// from the adapter's directory, a bare one from the repository root, and one naming only a skill means its
// SKILL.md.
func adapterTarget(root, rel, text string) (string, []string) {
	var out []string
	skill := ""
	seen := map[string]bool{}
	for _, loc := range skillLinkRe.FindAllStringSubmatchIndex(text, -1) {
		m := make([]string, 4)
		for i := range m {
			if loc[2*i] >= 0 {
				m[i] = text[loc[2*i]:loc[2*i+1]]
			}
		}
		if skill == "" {
			skill = m[2]
		}
		// A match cut short by a character the pattern does not take (SKILL.md~) names a different file.
		if end := loc[1]; end < len(text) && !strings.ContainsRune(" \t\r\n)]\"'`>,;:#?!*", rune(text[end])) &&
			!(text[end] == '.' && (end+1 == len(text) || strings.ContainsRune(" \t\r\n", rune(text[end+1])))) {
			out = append(out, fmt.Sprintf("%s: links to unsupported path %s", rel, strings.Fields(text[loc[0]:])[0]))
			continue
		}
		sub := strings.TrimRight(m[3], ".")
		if strings.Trim(sub, "/") == "" {
			sub = "/SKILL.md"
		}
		// Clean before the prefix check: a bare link keeps its ".." segments, which os.Stat would follow.
		target := path.Clean(".agents/skills/" + m[2] + sub)
		if m[1] != "" {
			target = path.Join(path.Dir(rel), m[1]+target)
		}
		// Key by the named skill too: the same target reached through another skill's .. must still fail.
		key := m[2] + "\x00" + target
		if seen[key] {
			continue
		}
		seen[key] = true
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(target)))
		if !strings.HasPrefix(target, ".agents/skills/"+m[2]+"/") || err != nil || !info.Mode().IsRegular() {
			out = append(out, fmt.Sprintf("%s: links to missing %s", rel, target))
		}
	}
	return skill, out
}

// SkillInventory lists the adopter skills (directories under .agents/skills/ that neither are canonical
// nor carry a canonical skill's checks through an alias) and the adapter aliases, for doctor's report.
func SkillInventory(root string) (adopter, aliases []string) {
	_, adopter, aliases = inspectSkills(root)
	return adopter, aliases
}

func checkSkillAdapters(root string) []string {
	out, _, _ := inspectSkills(root)
	return out
}

func inspectSkills(root string) (out, adopter, aliases []string) {
	adopter, aliases = []string{}, []string{}
	strict := map[string]bool{} // skills whose SKILL.md carries canonical checks
	isCanonical := map[string]bool{}
	for _, s := range canonicalSkills {
		isCanonical[s] = true
	}
	// alias records an adapter for name whose links resolve into another skill, and returns the skill
	// its first link names (name itself when it has no link).
	alias := func(adapterRel, adapterLabel, name, text string) string {
		skill, dangling := adapterTarget(root, adapterRel, text)
		out = append(out, dangling...)
		if skill == "" {
			return name
		}
		if skill != name && len(dangling) == 0 {
			aliases = append(aliases, fmt.Sprintf("alias: %s -> .agents/skills/%s", adapterLabel, skill))
		}
		return skill
	}

	for _, name := range canonicalSkills {
		// The generic file and each adapter's target carry the canonical checks; the Claude and Gemini
		// adapters may resolve to different skills, and then every target is checked.
		var targets []string
		addTarget := func(skill string) {
			for _, t := range targets {
				if t == skill {
					return
				}
			}
			targets = append(targets, skill)
		}
		// Codex and Antigravity read .agents/skills/ directly, so the generic file is checked even when
		// both adapters alias elsewhere.
		addTarget(name)

		claudeRel := ".claude/skills/" + name + "/SKILL.md"
		if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claudeRel))); err != nil {
			out = append(out, fmt.Sprintf("%s: missing Claude skill adapter", claudeRel))
		} else {
			text := string(data)
			skill := alias(claudeRel, ".claude/skills/"+name, name, text)
			addTarget(skill)
			wantLink := "../../../.agents/skills/" + skill + "/SKILL.md"
			if !strings.Contains(text, wantLink) || !strings.Contains(text, "fully") {
				out = append(out, fmt.Sprintf("%s: must link to %s and instruct to read it fully", claudeRel, wantLink))
			}
			lines := strings.Count(strings.TrimRight(text, "\n"), "\n") + 1
			if lines > maxSkillLines {
				out = append(out, fmt.Sprintf("%s: %d lines exceeds max %d lines", claudeRel, lines, maxSkillLines))
			}
			if strings.Contains(text, "- [ ]") || strings.Contains(text, "- [x]") {
				out = append(out, fmt.Sprintf("%s: contains copied checklist checkbox", claudeRel))
			}
		}

		geminiRel := ".gemini/commands/" + name + ".toml"
		if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(geminiRel))); err != nil {
			out = append(out, fmt.Sprintf("%s: missing Gemini command adapter", geminiRel))
		} else {
			text := string(data)
			skill := alias(geminiRel, geminiRel, name, text)
			addTarget(skill)
			wantLink := ".agents/skills/" + skill + "/SKILL.md"
			if !strings.Contains(text, wantLink) || !strings.Contains(text, "fully") {
				out = append(out, fmt.Sprintf("%s: must reference %s and instruct to read it fully", geminiRel, wantLink))
			}
		}

		for _, skill := range targets {
			strict[skill] = true
			canonRel := ".agents/skills/" + skill + "/SKILL.md"
			canonBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonRel)))
			if err != nil || len(strings.TrimSpace(string(canonBytes))) == 0 {
				out = append(out, fmt.Sprintf("%s: missing canonical skill", canonRel))
				continue
			}
			canonText := string(canonBytes)
			if !strings.Contains(canonText, "## 1.") {
				out = append(out, fmt.Sprintf("%s: missing numbered workflow steps (placeholder skill not allowed)", canonRel))
			}
			for _, marker := range requiredSkillMarkers[name] {
				if !strings.Contains(canonText, marker) {
					out = append(out, fmt.Sprintf("%s: missing required workflow procedure marker %q", canonRel, marker))
				}
			}
		}
	}

	// Adopter adapters are checked only for links that resolve.
	if entries, err := os.ReadDir(filepath.Join(root, ".claude", "skills")); err == nil {
		for _, e := range entries {
			if !e.IsDir() || isCanonical[e.Name()] {
				continue
			}
			rel := ".claude/skills/" + e.Name() + "/SKILL.md"
			if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
				alias(rel, ".claude/skills/"+e.Name(), e.Name(), string(data))
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".gemini", "commands")); err == nil {
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name(), ".toml")
			if e.IsDir() || !ok || isCanonical[name] {
				continue
			}
			rel := ".gemini/commands/" + e.Name()
			if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
				alias(rel, rel, name, string(data))
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".agents", "skills")); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strict[e.Name()] && !isCanonical[e.Name()] {
				adopter = append(adopter, e.Name())
			}
		}
	}
	return out, adopter, aliases
}

func checkHookConfigs(root string) []string {
	var out []string
	configs := []struct {
		rel   string
		agent string
	}{
		{rel: ".claude/settings.json", agent: "claude"},
		{rel: ".codex/hooks.json", agent: "codex"},
		{rel: ".gemini/settings.json", agent: "gemini"},
	}
	for _, cfg := range configs {
		rel := cfg.rel
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			out = append(out, fmt.Sprintf("%s: missing hook config file", rel))
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			out = append(out, fmt.Sprintf("%s: invalid JSON: %v", rel, err))
			continue
		}
		text := string(data)
		for _, ev := range requiredHooks {
			if !strings.Contains(text, "ateam hook --event "+ev) {
				out = append(out, fmt.Sprintf("%s: missing hook command for event %s", rel, ev))
			}
		}
		if !strings.Contains(text, "--agent "+cfg.agent) {
			out = append(out, fmt.Sprintf("%s: missing --agent %s in hook commands", rel, cfg.agent))
		}

		if rel == ".gemini/settings.json" && !strings.Contains(text, `"contextFileName": "AGENTS.md"`) {
			out = append(out, fmt.Sprintf("%s: missing contextFileName AGENTS.md", rel))
		}
	}
	out = append(out, checkAntigravityConfigs(root)...)
	return out
}

func checkAntigravityConfigs(root string) []string {
	var out []string
	// shortcut: Antigravity hooks.json omits pre-compact, add pre-compact hook when Antigravity exposes a compaction event
	agyRel := ".agents/hooks.json"
	if agyData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(agyRel))); err != nil {
		out = append(out, fmt.Sprintf("%s: missing hook config file", agyRel))
	} else {
		var parsed map[string]any
		if err := json.Unmarshal(agyData, &parsed); err != nil {
			out = append(out, fmt.Sprintf("%s: invalid JSON: %v", agyRel, err))
		} else {
			text := string(agyData)
			for _, ev := range []string{"session-start", "before-prompt", "turn-end"} {
				if !strings.Contains(text, "ateam hook --event "+ev) {
					out = append(out, fmt.Sprintf("%s: missing hook command for event %s", agyRel, ev))
				}
			}
			if !strings.Contains(text, "--agent agy") {
				out = append(out, fmt.Sprintf("%s: missing --agent agy in hook commands", agyRel))
			}
		}
	}

	for _, rel := range []string{".agents/plugins/ateam/plugin.json", ".agents/plugins/ateam/mcp_config.json"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			out = append(out, fmt.Sprintf("%s: missing Antigravity plugin file", rel))
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			out = append(out, fmt.Sprintf("%s: invalid JSON: %v", rel, err))
			continue
		}
		text := string(data)
		if rel == ".agents/plugins/ateam/plugin.json" && !strings.Contains(text, `"ateam"`) {
			out = append(out, fmt.Sprintf("%s: missing plugin name ateam", rel))
		}
		if rel == ".agents/plugins/ateam/mcp_config.json" && (!strings.Contains(text, `"ateam"`) || !strings.Contains(text, `"mcp"`)) {
			out = append(out, fmt.Sprintf("%s: missing ateam mcp server entry", rel))
		}
	}
	return out
}

// FormatStaleBinaryWarning formats the missing-provenance or behind-HEAD warning shared by doctor and session-start.
func FormatStaleBinaryWarning(shortBuild, headSHA string) string {
	shortBuild = strings.TrimSpace(shortBuild)
	if shortBuild == "" || shortBuild == "dev" || shortBuild == "unknown" {
		return "binary has no build provenance; build with task build"
	}
	return fmt.Sprintf("binary %s is behind HEAD %s; run task build", shortBuild, strings.TrimSpace(headSHA))
}

// DiskBuildStatus reports the ./bin/ateam build label for doctor, or "absent" / "unreadable",
// with a warning when a binary exists but its build cannot be read (#121).
func DiskBuildStatus(ctx context.Context, root string) (string, []string) {
	_, label := state.ReadDiskBinaryBuild(ctx, root)
	if label == "unreadable" {
		return label, []string{"./bin/ateam --version names no build or failed; the disk build is unknown, run task build"}
	}
	return label, nil
}

// ateamModule is the module path whose checkout the binary's build commit can be compared against.
// Go module paths are case-sensitive, so the match is exact.
const ateamModule = "github.com/c360studio/ateam"

// IsAteamCheckout reports whether root's go.mod declares the ateam module. It line-scans the file
// without spawning a subprocess, so the session-start hook stays inside its latency budget (#150).
func IsAteamCheckout(root string) bool {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module")
		if !ok || rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
			continue
		}
		return strings.Trim(strings.TrimSpace(rest), `"`) == ateamModule
	}
	return false
}

// BuildProvenanceNotApplicable is the doctor build_provenance value outside the ateam checkout.
const BuildProvenanceNotApplicable = "not applicable (binary is not built from this checkout)"

// BuildProvenanceStatus returns the doctor build_provenance field and the build warnings, reading
// go.mod once. Outside the ateam checkout the binary was never built from root, so there are no warnings (#150).
func BuildProvenanceStatus(ctx context.Context, root, buildRev, headSHA string) (string, []string) {
	if !IsAteamCheckout(root) {
		return BuildProvenanceNotApplicable, nil
	}
	return "checked", checkBuildProvenance(ctx, root, buildRev, headSHA)
}

// CheckBuildProvenance returns the build warnings of BuildProvenanceStatus.
func CheckBuildProvenance(ctx context.Context, root, buildRev, headSHA string) []string {
	_, warnings := BuildProvenanceStatus(ctx, root, buildRev, headSHA)
	return warnings
}

// checkBuildProvenance checks whether the running binary's commit is an ancestor of checkout HEAD.
func checkBuildProvenance(ctx context.Context, root, buildRev, headSHA string) []string {
	buildRev = strings.TrimSpace(buildRev)
	headSHA = strings.TrimSpace(headSHA)
	if buildRev == "" || buildRev == "dev" || buildRev == "unknown" {
		return []string{FormatStaleBinaryWarning(buildRev, headSHA)}
	}
	if headSHA == "" || headSHA == "uncommitted" {
		return nil
	}
	shortBuild := buildRev
	if len(shortBuild) > 7 {
		shortBuild = shortBuild[:7]
	}
	if state.SameCommitSHA(shortBuild, headSHA) {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", "merge-base", "--is-ancestor", buildRev, "HEAD")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return []string{fmt.Sprintf("binary build commit %s is not an ancestor of checkout HEAD %s; rebuild ./bin/ateam from this checkout", shortBuild, headSHA)}
	}
	return []string{FormatStaleBinaryWarning(shortBuild, headSHA)}
}

// CheckUnarchivedOpenSpecChanges returns a doctor warning for any OpenSpec change under openspec/changes/ whose tasks are all complete.
func CheckUnarchivedOpenSpecChanges(root string) []string {
	changes, err := state.ScanOpenSpecQueue(root)
	if err != nil {
		return nil
	}
	var warnings []string
	for _, ch := range changes {
		if ch.IsCompletedUnarchived() {
			warnings = append(warnings, fmt.Sprintf("openspec/changes/%s is completed (%d/%d tasks) and not archived; move to openspec/changes/archive/<date>-%s", ch.Name, ch.CompletedTasks, ch.TotalTasks, ch.Name))
		}
	}
	return warnings
}

// ParseRunningServeProcesses extracts PIDs of active `ateam serve` processes from `ps -eo pid=,args=` output.
func ParseRunningServeProcesses(psOutput string, selfPID int) []int {
	var pids []int
	seen := make(map[int]bool)
	for _, rawLine := range strings.Split(strings.ReplaceAll(psOutput, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 || pid == selfPID || seen[pid] {
			continue
		}
		base := filepath.Base(fields[1])
		if (base == "ateam" || base == "ateam.exe") && fields[2] == "serve" {
			seen[pid] = true
			pids = append(pids, pid)
		}
	}
	return pids
}

// CheckDuplicateServeProcesses returns a doctor warning when multiple `ateam serve` processes are running concurrently.
func CheckDuplicateServeProcesses(pids []int) []string {
	if len(pids) <= 1 {
		return nil
	}
	parts := make([]string, len(pids))
	for i, p := range pids {
		parts[i] = strconv.Itoa(p)
	}
	return []string{fmt.Sprintf("multiple ateam serve processes detected (%d running: PIDs %s); stop extra servers to prevent .ateam/state.json cache races", len(pids), strings.Join(parts, ", "))}
}

// InspectRunningServeProcesses queries OS processes and warns if duplicate `ateam serve` instances are running.
func InspectRunningServeProcesses(ctx context.Context) ([]int, []string) {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, "ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return nil, nil
	}
	pids := ParseRunningServeProcesses(string(out), os.Getpid())
	return pids, CheckDuplicateServeProcesses(pids)
}

func parsePSStartTime(tokens []string) (time.Time, string, bool) {
	if len(tokens) != 5 {
		return time.Time{}, "", false
	}
	raw := strings.Join(tokens, " ")
	t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", raw, time.Local)
	if err != nil {
		return time.Time{}, "", false
	}
	return t, raw, true
}

func canTrustRecordedMCPBuild(rec state.MCPProcess, lstartRaw string, psStart time.Time, hasPSStart bool) bool {
	if strings.TrimSpace(rec.BuildCommit) == "" {
		return false
	}
	if strings.TrimSpace(rec.StartToken) != "" && lstartRaw != "" && strings.TrimSpace(rec.StartToken) != lstartRaw {
		return false
	}
	if rec.StartedAt != nil && hasPSStart {
		diff := psStart.Sub(rec.StartedAt.In(time.Local))
		if diff < 0 {
			diff = -diff
		}
		if diff > 5*time.Second {
			return false
		}
	}
	return true
}

func parsePSProcessLine(rawLine string) (pid, ppid int, psStart time.Time, lstartRaw string, hasPSStart bool, cmdFields []string, ok bool) {
	line := strings.TrimSpace(rawLine)
	if line == "" {
		return 0, 0, time.Time{}, "", false, nil, false
	}
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return 0, 0, time.Time{}, "", false, nil, false
	}
	var err error
	pid, err = strconv.Atoi(fields[0])
	if err != nil || pid <= 0 {
		return 0, 0, time.Time{}, "", false, nil, false
	}
	cmdFields = fields[1:]
	if len(fields) >= 8 {
		if t, raw, startOK := parsePSStartTime(fields[1:6]); startOK {
			return pid, 0, t, raw, true, fields[6:], true
		}
	}
	if p, err := strconv.Atoi(fields[1]); err == nil && p >= 0 {
		ppid = p
		cmdFields = fields[2:]
		if len(fields) >= 9 {
			if t, raw, startOK := parsePSStartTime(fields[2:7]); startOK {
				return pid, ppid, t, raw, true, fields[7:], true
			}
		}
	}
	if len(cmdFields) == 0 {
		return 0, 0, time.Time{}, "", false, nil, false
	}
	return pid, ppid, time.Time{}, "", false, cmdFields, true
}

// ParseRunningMCPProcesses extracts active `ateam mcp` processes from `ps -eo pid=,ppid=,lstart=,args=` (or `pid=,lstart=,args=` / `pid=,args=`) output and joins recorded process metadata onto confirmed MCP processes.
func ParseRunningMCPProcesses(psOutput string, selfPID int, recorded []state.MCPProcess, fallbackBuild string) []state.MCPProcess {
	byPID := make(map[int]state.MCPProcess, len(recorded))
	for _, r := range recorded {
		if r.PID <= 0 || r.PID == selfPID {
			continue
		}
		byPID[r.PID] = r
	}
	fallbackBuild = strings.TrimSpace(fallbackBuild)
	if fallbackBuild == "" {
		fallbackBuild = "unknown"
	}
	var out []state.MCPProcess
	seen := make(map[int]bool)
	for _, rawLine := range strings.Split(strings.ReplaceAll(psOutput, "\r\n", "\n"), "\n") {
		pid, ppid, psStart, lstartRaw, hasPSStart, cmdFields, ok := parsePSProcessLine(rawLine)
		if !ok || pid == selfPID || seen[pid] || len(cmdFields) < 2 {
			continue
		}
		base := filepath.Base(cmdFields[0])
		if (base == "ateam" || base == "ateam.exe") && cmdFields[1] == "mcp" {
			seen[pid] = true
			if rec, ok := byPID[pid]; ok && canTrustRecordedMCPBuild(rec, lstartRaw, psStart, hasPSStart) {
				rec.PPID = ppid
				out = append(out, rec)
			} else {
				proc := state.MCPProcess{
					PID:         pid,
					PPID:        ppid,
					BuildCommit: fallbackBuild,
				}
				if hasPSStart {
					utc := psStart.UTC()
					proc.StartedAt = &utc
				}
				out = append(out, proc)
			}
		}
	}
	return out
}

func isShellWrapperComm(comm string) bool {
	switch strings.TrimPrefix(strings.ToLower(strings.TrimSpace(comm)), "-") {
	case "sh", "bash", "zsh", "dash", "fish", "ksh", "csh", "tcsh":
		return true
	default:
		return false
	}
}

// FilterClientMCPProcesses returns running `ateam mcp` processes that belong to clientPID's process tree (including when clientPID is a shell wrapper around the hook).
func FilterClientMCPProcesses(psOutput string, selfPID, clientPID int, recorded []state.MCPProcess, fallbackBuild string) []state.MCPProcess {
	if clientPID <= 1 {
		return nil
	}
	parentOf := make(map[int]int)
	commOf := make(map[int]string)
	for _, rawLine := range strings.Split(strings.ReplaceAll(psOutput, "\r\n", "\n"), "\n") {
		pid, ppid, _, _, _, cmdFields, ok := parsePSProcessLine(rawLine)
		if !ok {
			continue
		}
		parentOf[pid] = ppid
		commOf[pid] = filepath.Base(cmdFields[0])
	}
	allMCP := ParseRunningMCPProcesses(psOutput, selfPID, recorded, fallbackBuild)
	if len(allMCP) == 0 {
		return nil
	}
	descendantsOf := func(rootPID int) []state.MCPProcess {
		if rootPID <= 1 {
			return nil
		}
		var matched []state.MCPProcess
		maxHops := len(parentOf) + 2
		for _, p := range allMCP {
			cur := p.PID
			for hops := 0; cur > 1 && hops < maxHops; hops++ {
				if cur == rootPID {
					matched = append(matched, p)
					break
				}
				next := parentOf[cur]
				if next == 0 && cur == p.PID {
					next = p.PPID
				}
				cur = next
			}
		}
		return matched
	}
	if matched := descendantsOf(clientPID); len(matched) > 0 {
		return matched
	}
	if clientPID == selfPID || isShellWrapperComm(commOf[clientPID]) {
		if parentPID := parentOf[clientPID]; parentPID > 1 {
			return descendantsOf(parentPID)
		}
	}
	return nil
}

// mcpRepoOther is the MCPProcess.Repo value of a process that serves another repository (#151).
const mcpRepoOther = "other"

// mcpProcessCwd reads a process's working directory; tests replace it.
var mcpProcessCwd = readProcessCwd

// listProcesses and loadLiveMCPProcesses feed the MCP inspectors; tests replace them.
var (
	listProcesses        = psProcessList
	loadLiveMCPProcesses = state.LoadLiveMCPProcesses
)

// psProcessList returns `ps -eo pid=,ppid=,lstart=,args=` output.
func psProcessList(ctx context.Context) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "ps", "-eo", "pid=,ppid=,lstart=,args=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	return cmd.Output()
}

// readProcessCwd returns pid's working directory, or false when it cannot be read. macOS has no
// /proc, so it spawns lsof (about 20ms); callers run it only for unregistered MCP processes.
func readProcessCwd(ctx context.Context, pid int) (string, bool) {
	switch runtime.GOOS {
	case "linux":
		dir, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
		// A removed working directory reads as "<path> (deleted)"; treat it as unreadable.
		return dir, err == nil && dir != "" && !strings.HasSuffix(dir, " (deleted)")
	case "darwin":
		cctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		out, err := exec.CommandContext(cctx, "lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
		if err != nil {
			return "", false
		}
		for _, line := range strings.Split(string(out), "\n") {
			if dir, ok := strings.CutPrefix(line, "n"); ok && dir != "" {
				return dir, true
			}
		}
	}
	return "", false
}

// attributeMCPProcesses marks procs that serve another repository with Repo "other": a process is
// this root's when this root's registry lists it or its working directory is in this repository.
// A process whose working directory cannot be read stays this root's, so it still warns (#151).
func attributeMCPProcesses(ctx context.Context, root string, procs, recorded []state.MCPProcess) {
	ours := make(map[int]bool, len(recorded))
	for _, r := range recorded {
		ours[r.PID] = true
	}
	for i := range procs {
		if ours[procs[i].PID] {
			continue
		}
		if cwd, ok := mcpProcessCwd(ctx, procs[i].PID); ok && !state.InRepository(root, cwd) {
			procs[i].Repo = mcpRepoOther
		}
	}
}

// CheckClientMCPProcesses formats one-line session hook warnings for any client-tree `ateam mcp` process whose build is unknown or older than headSHA.
func CheckClientMCPProcesses(procs []state.MCPProcess, headSHA string) []string {
	headSHA = strings.TrimSuffix(strings.TrimSpace(headSHA), "-dirty")
	if headSHA == "" || headSHA == "unknown" || headSHA == "dev" || headSHA == "uncommitted" {
		return nil
	}
	var warnings []string
	for _, p := range procs {
		if p.Repo == mcpRepoOther {
			continue
		}
		procRev := strings.TrimSuffix(strings.TrimSpace(p.BuildCommit), "-dirty")
		fullRev := strings.TrimSuffix(strings.TrimSpace(p.FullCommit), "-dirty")
		sameCommit := state.SameCommitSHA(procRev, headSHA) || (fullRev != "" && state.SameCommitSHA(fullRev, headSHA))
		if procRev == "" || procRev == "unknown" || procRev == "dev" || !sameCommit {
			label := strings.TrimSpace(p.BuildCommit)
			if label == "" {
				label = "unknown"
			}
			warnings = append(warnings, fmt.Sprintf("MCP server PID %d (build %s) is older than HEAD; restart the client", p.PID, label))
		}
	}
	return warnings
}

// InspectClientMCPProcesses enumerates `ateam mcp` processes under clientPID's process tree and returns one-line warnings for any whose build commit is unknown or older than headSHA.
func InspectClientMCPProcesses(ctx context.Context, root, headSHA string, clientPID int) ([]state.MCPProcess, []string) {
	recorded := loadLiveMCPProcesses(root)
	out, err := listProcesses(ctx)
	if err != nil {
		return nil, nil
	}
	procs := FilterClientMCPProcesses(string(out), os.Getpid(), clientPID, recorded, "unknown")
	attributeMCPProcesses(ctx, root, procs, recorded)
	return procs, CheckClientMCPProcesses(procs, headSHA)
}

// CheckRunningMCPProcesses returns doctor warnings for any running `ateam mcp` process whose build commit differs from expectedCommit.
func CheckRunningMCPProcesses(procs []state.MCPProcess, expectedCommit string) []string {
	expectedCommit = strings.TrimSuffix(strings.TrimSpace(expectedCommit), "-dirty")
	if expectedCommit == "" || expectedCommit == "unknown" || expectedCommit == "dev" || expectedCommit == "uncommitted" {
		return nil
	}
	var warnings []string
	for _, p := range procs {
		if p.Repo == mcpRepoOther {
			continue
		}
		procRev := strings.TrimSuffix(strings.TrimSpace(p.BuildCommit), "-dirty")
		if procRev == "" || procRev == "unknown" || procRev == "dev" || !state.SameCommitSHA(procRev, expectedCommit) {
			label := strings.TrimSpace(p.BuildCommit)
			if label == "" {
				label = "unknown"
			}
			warnings = append(warnings, fmt.Sprintf("running ateam mcp process (PID %d, build %s) is older than %s; restart the MCP client", p.PID, label, expectedCommit))
		}
	}
	return warnings
}

// InspectRunningMCPProcesses queries recorded and OS `ateam mcp` processes and warns if any are behind checkout HEAD or on-disk binary.
func InspectRunningMCPProcesses(ctx context.Context, root, headSHA string) ([]state.MCPProcess, []string) {
	recorded := loadLiveMCPProcesses(root)
	diskRev, _ := state.ReadDiskBinaryBuild(ctx, root)
	out, err := listProcesses(ctx)
	if err != nil {
		return nil, []string{fmt.Sprintf("unable to inspect running ateam mcp processes via ps: %v", err)}
	}
	procs := ParseRunningMCPProcesses(string(out), os.Getpid(), recorded, "unknown")
	attributeMCPProcesses(ctx, root, procs, recorded)
	// Only the ateam checkout's HEAD says anything about an MCP server's build (#151); elsewhere
	// the comparison falls back to the on-disk binary as it does without a HEAD (#116, #121).
	expected := strings.TrimSpace(headSHA)
	if !IsAteamCheckout(root) || expected == "" || expected == "uncommitted" {
		expected = diskRev
	}
	return procs, CheckRunningMCPProcesses(procs, expected)
}
