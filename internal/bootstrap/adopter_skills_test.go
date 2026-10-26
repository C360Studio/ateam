package bootstrap

import (
	"context"
	"encoding/json"
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
		{"first link names the adapter's skill",
			map[string]string{".claude/skills/qp/SKILL.md": "Read [.agents/skills/query-pattern/SKILL.md](../../../.agents/skills/semteams-preflight/SKILL.md).\n"},
			[]string{"query-pattern"},
			[]string{claudeAlias, "alias: .claude/skills/qp -> .agents/skills/query-pattern"}},
		{"trailing dot ends the skill name",
			map[string]string{".claude/skills/qp/SKILL.md": "See .agents/skills/query-pattern.\n"},
			[]string{"query-pattern"},
			[]string{claudeAlias, "alias: .claude/skills/qp -> .agents/skills/query-pattern"}},
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
		dirs   []string
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
		{name: "both adapters alias and the generic skill is a placeholder",
			files: map[string]string{".gemini/commands/preflight.toml": "prompt = \"Read .agents/skills/semteams-preflight/SKILL.md fully.\"\n", ".agents/skills/preflight/SKILL.md": "TODO\n"},
			want:  []string{".agents/skills/preflight/SKILL.md: missing numbered workflow steps"}},
		{name: "both adapters alias and the generic skill is missing",
			files:  map[string]string{".gemini/commands/preflight.toml": "prompt = \"Read .agents/skills/semteams-preflight/SKILL.md fully.\"\n"},
			remove: ".agents/skills/preflight/SKILL.md",
			want:   []string{".agents/skills/preflight/SKILL.md: missing canonical skill"}},
		{name: "link names a skill directory without SKILL.md",
			files: map[string]string{".claude/skills/x/SKILL.md": "Read .agents/skills/emptydir fully.\n"},
			dirs:  []string{".agents/skills/emptydir"},
			want:  []string{".claude/skills/x/SKILL.md: links to missing .agents/skills/emptydir/SKILL.md"}},
		{name: "link names a directory inside a skill",
			files: map[string]string{".claude/skills/x/SKILL.md": "Read .agents/skills/query-pattern/refs fully.\n"},
			dirs:  []string{".agents/skills/query-pattern/refs"},
			want:  []string{".claude/skills/x/SKILL.md: links to missing .agents/skills/query-pattern/refs"}},
		{name: "relative link with the wrong depth",
			files: map[string]string{".claude/skills/x/SKILL.md": "Read [it](../.agents/skills/query-pattern/SKILL.md) fully.\n"},
			want:  []string{".claude/skills/x/SKILL.md: links to missing .claude/skills/.agents/skills/query-pattern/SKILL.md"}},
		{name: "bare link escapes its skill with ..",
			files: map[string]string{".claude/skills/qp/SKILL.md": "Read .agents/skills/query-pattern/../../../AGENTS.md fully.\n"},
			want:  []string{".claude/skills/qp/SKILL.md: links to missing AGENTS.md"}},
		{name: "bare link crosses into another skill with ..",
			files: map[string]string{".claude/skills/qp/SKILL.md": "Read .agents/skills/query-pattern/../semteams-preflight/SKILL.md fully.\n"},
			want:  []string{".claude/skills/qp/SKILL.md: links to missing .agents/skills/semteams-preflight/SKILL.md"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := skillsBrownfield(t)
			writeFiles(t, dir, c.files)
			for _, d := range c.dirs {
				if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(d)), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if c.remove != "" {
				if err := os.Remove(filepath.Join(dir, filepath.FromSlash(c.remove))); err != nil {
					t.Fatal(err)
				}
			}
			requireViolation(t, CheckParity(dir), c.want...)
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter skills and adapter aliases
func TestSkillInventoryEmptyListsMarshalAsArrays(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	adopter, aliases := SkillInventory(dir)
	got, err := json.Marshal(map[string]any{"adopter_skills": adopter, "skill_aliases": aliases})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"adopter_skills":[],"skill_aliases":[]}`; string(got) != want {
		t.Fatalf("doctor inventory JSON = %s, want %s", got, want)
	}
}
