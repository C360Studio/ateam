package bootstrap

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const semteamsPreflight = `---
name: semteams-preflight
---

# SemTeams preflight

## 1. Pick the gates

Run task verify before pushing.

## 2. Record Evidence

Paste the gate result lines.
`

const preflightAlias = `---
name: preflight
---

# Preflight

Read [.agents/skills/semteams-preflight/SKILL.md](../../../.agents/skills/semteams-preflight/SKILL.md) fully.
`

// skillsBrownfield is the #158 brownfield fixture plus an adopter skill with prose headings and no
// adapters, and a Claude preflight adapter aliasing the adopter's own preflight.
func skillsBrownfield(t *testing.T) string {
	t.Helper()
	rows := []int{2, 1, 3, 1, 2}
	dir := brownfield(t, adopterRules("### ", rows), adopterRules("## ", rows), adopterProtocol)
	writeFiles(t, dir, map[string]string{
		".agents/skills/query-pattern/SKILL.md":      "# Query patterns\n\n## When to use\n\nWhen a query repeats.\n\n## Shape\n\nProse, not steps.\n",
		".agents/skills/semteams-preflight/SKILL.md": semteamsPreflight,
		".claude/skills/preflight/SKILL.md":          preflightAlias,
	})
	return dir
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter skills and adapter aliases
func TestAdopterSkillsAndAliasesPass(t *testing.T) {
	const claudeAlias = "alias: .claude/skills/preflight -> .agents/skills/semteams-preflight"
	cases := []struct {
		name        string
		files       map[string]string
		wantAdopter []string
		wantAliases []string
	}{
		{"adopter skill and Claude preflight alias", nil,
			[]string{"query-pattern"}, []string{claudeAlias}},
		{"Gemini preflight aliases a third skill",
			map[string]string{
				".agents/skills/gem-preflight/SKILL.md": semteamsPreflight,
				".gemini/commands/preflight.toml":       "prompt = \"Read .agents/skills/gem-preflight/SKILL.md fully.\"\n",
			},
			[]string{"query-pattern"},
			[]string{claudeAlias, "alias: .gemini/commands/preflight.toml -> .agents/skills/gem-preflight"}},
		{"adopter adapters link and alias",
			map[string]string{
				".claude/skills/query-pattern/SKILL.md": "See .agents/skills/query-pattern/SKILL.md.\n",
				".claude/skills/qp/SKILL.md":            "See [it](../../../.agents/skills/query-pattern/SKILL.md).\n",
				".gemini/commands/qp.toml":              "prompt = \"Read .agents/skills/query-pattern/SKILL.md\"\n",
				".claude/skills/notes/SKILL.md":         "No link at all, prose only.\n",
			},
			[]string{"query-pattern"},
			[]string{claudeAlias, "alias: .claude/skills/qp -> .agents/skills/query-pattern", "alias: .gemini/commands/qp.toml -> .agents/skills/query-pattern"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := skillsBrownfield(t)
			writeFiles(t, dir, c.files)
			if v := CheckParity(dir); len(v) > 0 {
				t.Fatalf("CheckParity:\n- %s", strings.Join(v, "\n- "))
			}
			adopter, aliases := SkillInventory(dir)
			if !slices.Equal(adopter, c.wantAdopter) {
				t.Errorf("adopter skills = %q, want %q", adopter, c.wantAdopter)
			}
			if !slices.Equal(aliases, c.wantAliases) {
				t.Errorf("aliases = %q, want %q", aliases, c.wantAliases)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter skills and adapter aliases
func TestAdopterSkillsSensitivity(t *testing.T) {
	cases := []struct {
		name   string
		files  map[string]string
		remove string
		want   []string
	}{
		{name: "canonical skill missing ## 1.",
			files: map[string]string{".agents/skills/handoff/SKILL.md": "# Handoff\n\nRun ateam state handoff. Goal: hand off.\n"},
			want:  []string{".agents/skills/handoff/SKILL.md", "missing numbered workflow steps"}},
		{name: "canonical skill missing its Gemini adapter",
			remove: ".gemini/commands/pickup.toml",
			want:   []string{".gemini/commands/pickup.toml", "missing Gemini command adapter"}},
		{name: "canonical adapter without a link",
			files: map[string]string{".claude/skills/ponytail/SKILL.md": "Read the ponytail skill fully.\n"},
			want:  []string{".claude/skills/ponytail/SKILL.md", "must link to ../../../.agents/skills/ponytail/SKILL.md"}},
		{name: "Claude adapter links to a missing target",
			files: map[string]string{".claude/skills/query-pattern/SKILL.md": "Read [it](../../../.agents/skills/query-patterns/SKILL.md) fully.\n"},
			want:  []string{".claude/skills/query-pattern/SKILL.md: links to missing .agents/skills/query-patterns/SKILL.md"}},
		{name: "Gemini adapter links to a missing target",
			files: map[string]string{".gemini/commands/qp.toml": "prompt = \"Read .agents/skills/qp/SKILL.md fully.\"\n"},
			want:  []string{".gemini/commands/qp.toml: links to missing .agents/skills/qp/SKILL.md"}},
		{name: "alias target lacks the preflight markers",
			files: map[string]string{".agents/skills/semteams-preflight/SKILL.md": strings.ReplaceAll(semteamsPreflight, "Record Evidence", "Notes")},
			want:  []string{".agents/skills/semteams-preflight/SKILL.md", `"Record Evidence"`}},
		{name: "Gemini alias target lacks the preflight markers",
			files: map[string]string{
				".agents/skills/gem-preflight/SKILL.md": "# Gem\n\n## 1. Run\n\nRun task verify.\n",
				".gemini/commands/preflight.toml":       "prompt = \"Read .agents/skills/gem-preflight/SKILL.md fully.\"\n",
			},
			want: []string{".agents/skills/gem-preflight/SKILL.md", `"Record Evidence"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := skillsBrownfield(t)
			writeFiles(t, dir, c.files)
			if c.remove != "" {
				if err := os.Remove(filepath.Join(dir, filepath.FromSlash(c.remove))); err != nil {
					t.Fatal(err)
				}
			}
			requireViolation(t, CheckParity(dir), c.want...)
		})
	}
}
