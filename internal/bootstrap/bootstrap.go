// Package bootstrap implements ateam init, ateam doctor, and multi-agent setup parity verification.
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c360studio/ateam/internal/state"
)

const (
	maxAgentsMDBytes = 24000
	maxClaudeMDLines = 10
	maxSkillLines    = 25
)

var (
	canonicalSkills = []string{"pickup", "handoff", "preflight", "ponytail"}
	requiredHooks   = []string{"session-start", "before-prompt", "pre-compact", "turn-end"}
	expectedDomains = []string{
		"1. Ponytail decision ladder and code discipline",
		"2. Hooks, MCP budget, and multi-agent parity",
		"3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done",
	}
)

// Options configures InitRepo.
type Options struct {
	Force          bool
	BootstrapForge bool
	ForgeOverride  string // "", "github", "gitea"
}

// InitRepo idempotently scaffolds universal skills, hooks, MCP configs, and OpenSpec into root.
func InitRepo(ctx context.Context, root string, opts Options) ([]string, error) {
	var written []string
	files := defaultTemplateFiles()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		preserve := !opts.Force || isClientConfig(rel)
		if preserve {
			if _, err := os.Lstat(abs); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return written, fmt.Errorf("inspect %s: %w", rel, err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return written, fmt.Errorf("mkdir %s: %w", rel, err)
		}
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if preserve {
			flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
		}
		f, err := os.OpenFile(abs, flags, 0o644)
		if err != nil {
			return written, fmt.Errorf("create %s: %w", rel, err)
		}
		_, writeErr := f.WriteString(content)
		closeErr := f.Close()
		if writeErr != nil {
			return written, fmt.Errorf("write %s: %w", rel, writeErr)
		}
		if closeErr != nil {
			return written, fmt.Errorf("close %s: %w", rel, closeErr)
		}
		written = append(written, rel)
	}

	if err := ensureGitignore(root); err != nil {
		return written, err
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
			return written, err
		}
	}
	return written, nil
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
		if !strings.Contains(proto, "## 1. Hybrid State Architecture") || !strings.Contains(proto, "## 2. Work Lifecycle & Definition of Done") {
			out = append(out, ".agents/protocol.md: missing canonical protocol sections (must not be a placeholder stub)")
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
	agentsCounts := parseRuleSectionCounts(string(agentsData), "### ")
	indexCounts := parseRuleSectionCounts(string(indexData), "## ")
	if len(agentsCounts) != len(expectedDomains) || len(indexCounts) != len(expectedDomains) {
		return append(out, fmt.Sprintf("AGENTS.md (%d sections) and .agents/rules-index.md (%d sections) must both have %d numbered rule sections",
			len(agentsCounts), len(indexCounts), len(expectedDomains)))
	}
	for i, wantTitle := range expectedDomains {
		a, b := agentsCounts[i], indexCounts[i]
		if a.title != wantTitle || b.title != wantTitle {
			out = append(out, fmt.Sprintf("rule section %d: AGENTS.md = %q, .agents/rules-index.md = %q, want %q", i+1, a.title, b.title, wantTitle))
			continue
		}
		if a.rows == 0 || a.rows != b.rows {
			out = append(out, fmt.Sprintf("AGENTS.md and .agents/rules-index.md disagree on row count in %q: AGENTS.md has %d, .agents/rules-index.md has %d",
				wantTitle, a.rows, b.rows))
		}
	}
	return out
}

func parseRuleSectionCounts(text, prefix string) []sectionCount {
	var sections []sectionCount
	cur := -1
	for _, line := range strings.Split(text, "\n") {
		if title, ok := strings.CutPrefix(line, prefix); ok && len(title) >= 3 && title[0] >= '1' && title[0] <= '9' && title[1] == '.' {
			sections = append(sections, sectionCount{title: strings.TrimSpace(title)})
			cur = len(sections) - 1
			continue
		}
		if cur >= 0 && strings.HasPrefix(line, "## ") && prefix != "## " {
			cur = -1
			continue
		}
		if cur < 0 || !strings.HasPrefix(line, "| ") {
			continue
		}
		if strings.HasPrefix(line, "| ---") || strings.HasPrefix(line, "| Gate |") {
			continue
		}
		sections[cur].rows++
	}
	return sections
}

func discoverSkills(root string) []string {
	seen := make(map[string]bool, len(canonicalSkills))
	out := make([]string, 0, len(canonicalSkills))
	for _, s := range canonicalSkills {
		out = append(out, s)
		seen[s] = true
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".agents", "skills")); err == nil {
		for _, e := range entries {
			if e.IsDir() && !seen[e.Name()] {
				out = append(out, e.Name())
				seen[e.Name()] = true
			}
		}
	}
	return out
}

var requiredSkillMarkers = map[string][]string{
	"handoff":   {"## 1.", "ateam state handoff", "Goal:"},
	"pickup":    {"## 1.", "ateam state pickup", "Verify Write Ownership"},
	"ponytail":  {"## 1.", "6-Step Decision Ladder", "// shortcut:"},
	"preflight": {"## 1.", "task verify", "Record Evidence"},
}

func checkSkillAdapters(root string) []string {
	var out []string
	for _, name := range discoverSkills(root) {
		canonRel := ".agents/skills/" + name + "/SKILL.md"
		canonBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonRel)))
		if err != nil || len(strings.TrimSpace(string(canonBytes))) == 0 {
			out = append(out, fmt.Sprintf("%s: missing canonical skill", canonRel))
		} else {
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

		claudeRel := ".claude/skills/" + name + "/SKILL.md"
		if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claudeRel))); err != nil {
			out = append(out, fmt.Sprintf("%s: missing Claude skill adapter", claudeRel))
		} else {
			text := string(data)
			wantLink := "../../../.agents/skills/" + name + "/SKILL.md"
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
			wantLink := ".agents/skills/" + name + "/SKILL.md"
			if !strings.Contains(text, wantLink) || !strings.Contains(text, "fully") {
				out = append(out, fmt.Sprintf("%s: must reference %s and instruct to read it fully", geminiRel, wantLink))
			}
		}
	}
	return out
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
	return out
}
