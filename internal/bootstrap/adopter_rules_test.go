package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const adopterProtocol = `# How we work

## Intake

Before touching state, agree the Deliverable, the Scope and non-goals, the Definition of Done and the Stop boundary.

## Handoff

End a session with ` + "`ateam state handoff`" + `.

## Landing

Archive the change under openspec/changes/archive/ before marking the PR ready.
`

// adopterRules renders a project-named rules file; sections are title -> row count.
func adopterRules(prefix string, rows []int) string {
	titles := []string{"1. Billing safety", "2. Ledger changes", "3. Release train", "4. Data retention", "5. On-call"}
	var b strings.Builder
	b.WriteString("# Rules\n\n")
	for i, title := range titles {
		b.WriteString(prefix + title + "\n\n| Gate | Rule |\n| --- | --- |\n")
		for r := 0; r < rows[i]; r++ {
			b.WriteString("| `review` | rule | where |\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func brownfield(t *testing.T, agents, index, protocol string) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	for rel, body := range map[string]string{"AGENTS.md": agents, ".agents/rules-index.md": index, ".agents/protocol.md": protocol} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestAgentSetupParity (parity_test.go) covers the ateam repository's own files.

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestAdopterRulesParityAcceptsProjectOwnedFiles(t *testing.T) {
	rows := []int{2, 1, 3, 1, 2}
	// The appendix table is in AGENTS.md only: it agrees only if the non-numbered heading ends "5. On-call".
	agents := adopterRules("### ", rows) + "## Appendix\n\n| not | counted |\n"
	dir := brownfield(t, agents, adopterRules("## ", rows), adopterProtocol)
	if v := CheckParity(dir); len(v) > 0 {
		t.Fatalf("brownfield files failed CheckParity:\n- %s", strings.Join(v, "\n- "))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestAdopterRulesParityDisagreement(t *testing.T) {
	rows := []int{2, 1, 3, 1, 2}
	agents := adopterRules("### ", rows)
	cases := []struct {
		name   string
		agents string
		index  string
		want   []string
	}{
		{"title differs", agents, strings.Replace(adopterRules("## ", rows), "3. Release train", "3. Release trains", 1), []string{"rule section 3", "AGENTS.md", ".agents/rules-index.md", "3. Release train"}},
		{"row count differs", agents, adopterRules("## ", []int{2, 1, 4, 1, 2}), []string{"row count", "3. Release train", "AGENTS.md has 3", ".agents/rules-index.md has 4"}},
		{"section missing from index", agents, strings.Replace(adopterRules("## ", rows), "5. On-call", "On-call", 1), []string{"5. On-call", "missing from .agents/rules-index.md"}},
		{"compact rows differ", "### 1. Safety\n\n|Gate|Rule|\n|---|---|\n|a|b|\n",
			"## 1. Safety\n\n|Gate|Rule|\n|---|---|\n|a|b|\n|c|d|\n|e|f|\n",
			[]string{"row count", "1. Safety", "AGENTS.md has 1", ".agents/rules-index.md has 3"}},
		{"zero-row section", "### 1. Safety\n\n- a rule written as a list\n",
			"## 1. Safety\n\n- a rule written as a list\n",
			[]string{"AGENTS.md: rule section \"1. Safety\" has no table rows", ".agents/rules-index.md: rule section \"1. Safety\" has no table rows"}},
		// Picking the larger ## prose level would discard the ### rules and certify the 1-versus-2 mismatch.
		{"## and ### both own rows, rules disagree", "# Project\n\n## 1. Commands\n\n| Command | Purpose |\n| --- | --- |\n| build | compile |\n| test | verify |\n| fmt | format |\n\n### 1. Safety\n\n| Gate | Rule |\n| --- | --- |\n| review | validate |\n", "# Project\n\n## 1. Commands\n\n| Command | Purpose |\n| --- | --- |\n| build | compile |\n| test | verify |\n| fmt | format |\n\n### 1. Safety\n\n| Gate | Rule |\n| --- | --- |\n| review | validate |\n| verify | preserve data |\n",
			[]string{"AGENTS.md: both ## N. and ### N. headings own table rows", ".agents/rules-index.md: both ## N. and ### N. headings own table rows"}},
		{"## and ### both own rows, rules agree", "# Project\n\n## 1. Commands\n\n| Command | Purpose |\n| --- | --- |\n| build | compile |\n| test | verify |\n| fmt | format |\n\n### 1. Safety\n\n| Gate | Rule |\n| --- | --- |\n| review | validate |\n", "# Project\n\n## 1. Commands\n\n| Command | Purpose |\n| --- | --- |\n| build | compile |\n| test | verify |\n| fmt | format |\n\n### 1. Safety\n\n| Gate | Rule |\n| --- | --- |\n| review | validate |\n",
			[]string{"AGENTS.md: both ## N. and ### N. headings own table rows", "number only the rule section headings"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := brownfield(t, c.agents, c.index, adopterProtocol)
			requireViolation(t, CheckParity(dir), c.want...)
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestAdopterRulesParityAcceptsAdopterLayouts(t *testing.T) {
	index := "# Index\n\n## 1. Safety\n\n| Rule | Enforced by |\n| :--- | ---: |\n| a | b |\n| c | d |\n\n## 2. Data\n\n|Gate|Rule|\n|:-:|:-:|\n|e|f|\n"
	cases := []struct{ name, agents string }{
		{"numbered ## headings above ### rule sections",
			"# Project\n\n## 1. Purpose\n\nWhy.\n\n## 2. Rules\n\n### 1. Safety\n\n| Gate | Rule |\n| --- | --- |\n| a | b |\n| c | d |\n\n### 2. Data\n\n| Gate | Rule |\n| --- | --- |\n| e | f |\n"},
		{"indented compact rows",
			"### 1. Safety\n\n  |Gate|Rule|\n  |---|---|\n  |a|b|\n  |c|d|\n\n### 2. Data\n\n|x|y|\n|-|-|\n|e|f|\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if v := CheckParity(brownfield(t, c.agents, index, adopterProtocol)); len(v) > 0 {
				t.Fatalf("CheckParity:\n- %s", strings.Join(v, "\n- "))
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestParseRuleSectionCounts(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		want  []sectionCount
		mixed bool
	}{
		{"custom header and aligned separator",
			"## 1. Safety\n\n| Rule | Enforced by |\n| :--- | ---: |\n| a | b |\n| c | d |\n",
			[]sectionCount{{"1. Safety", 2}}, false},
		{"compact and indented rows",
			"### 1. Safety\n\n |Gate|Rule|\n |---|---|\n |a|b|\n|c|d|\n|e|f|\n",
			[]sectionCount{{"1. Safety", 3}}, false},
		{"rows without a separator all count",
			"## 1. Safety\n\n| a | b |\n| c | d |\n",
			[]sectionCount{{"1. Safety", 2}}, false},
		{"non-numbered heading ends a section",
			"### 1. Safety\n\n| a | b |\n\n## Appendix\n\n| x | y |\n| z | w |\n\n### Notes\n\n| q | r |\n",
			[]sectionCount{{"1. Safety", 1}}, false},
		{"numbered ## headings without rows are ignored",
			"## 1. Purpose\n\nWhy.\n\n## 2. Rules\n\n### 1. Safety\n\n| a | b |\n| c | d |\n\n### 2. Data\n\n| e | f |\n",
			[]sectionCount{{"1. Safety", 2}, {"2. Data", 1}}, false},
		{"## and ### both owning rows is mixed",
			"## 1. Purpose\n\n| a | b |\n| c | d |\n\n## 2. Rules\n\n### 1. Safety\n\n| e | f |\n",
			nil, true},
		{"only ## owns rows keeps ##",
			"## 1. Safety\n\n| a | b |\n\n### 1. Note\n\nprose\n",
			[]sectionCount{{"1. Safety", 1}}, false},
		{"no rows anywhere names the ### sections",
			"## 1. Purpose\n\n## 2. Rules\n\n### 1. Safety\n",
			[]sectionCount{{"1. Safety", 0}}, false},
		{"an all-blank row is a row, not a separator",
			"## 1. Safety\n\n| G | R |\n|---|---|\n| a | b |\n| | |\n| c | d |\n",
			[]sectionCount{{"1. Safety", 3}}, false},
		{"zero-row sections are kept",
			"## 1. Safety\n\n- a list\n\n## 2. Data\n",
			[]sectionCount{{"1. Safety", 0}, {"2. Data", 0}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, mixed := parseRuleSectionCounts(c.text)
			if mixed != c.mixed {
				t.Fatalf("mixed = %v, want %v", mixed, c.mixed)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestAdopterProtocolMarkers(t *testing.T) {
	rows := []int{1, 1, 1, 1, 1}
	agents, index := adopterRules("### ", rows), adopterRules("## ", rows)
	for _, c := range []struct{ marker, want string }{
		{"Deliverable", "Clarify step"},
		{"non-goals", "Clarify step"},
		{"Definition of Done", "Clarify step"},
		{"Stop boundary", "Clarify step"},
		{"ateam state handoff", "session handoff"},
		{"openspec/changes/archive/", "Close step"},
	} {
		t.Run(c.marker, func(t *testing.T) {
			proto := strings.ReplaceAll(adopterProtocol, c.marker, "removed")
			v := CheckParity(brownfield(t, agents, index, proto))
			requireViolation(t, v, ".agents/protocol.md", `"`+c.marker+`"`, c.want)
		})
	}
	t.Run("all Clarify fields named", func(t *testing.T) {
		v := CheckParity(brownfield(t, agents, index, "# Protocol\n\nClarify, Claim, Merge, Close. ateam state handoff openspec/changes/archive/\n"))
		requireViolation(t, v, `"Deliverable"`, `"non-goals"`, `"Definition of Done"`, `"Stop boundary"`)
	})
}

// spec: openspec/specs/harness/spec.md#Requirement: Opt-in design track
func TestInitWritesDesignRulesMatchingThisRepository(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(root, "openspec", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	own, err := os.ReadFile(filepath.Join(repoRoot(t), "openspec", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rules := func(data []byte) string {
		_, after, ok := strings.Cut(string(data), "\nrules:\n")
		if !ok {
			t.Fatalf("no rules: block in:\n%s", data)
		}
		return after
	}
	if got, want := rules(written), rules(own); got != want {
		t.Fatalf("init design rules drifted from openspec/config.yaml:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if !strings.HasPrefix(string(written), "schema: spec-driven\n") {
		t.Fatalf("config lost its schema line:\n%s", written)
	}
}
