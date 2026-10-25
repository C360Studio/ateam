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

Before touching state, agree the Deliverable, the Scope, the Definition of Done and the Stop boundary.

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
	b.WriteString("## Appendix\n\n| not | counted |\n")
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
	dir := brownfield(t, adopterRules("### ", rows), adopterRules("## ", rows), adopterProtocol)
	if v := CheckParity(dir); len(v) > 0 {
		t.Fatalf("brownfield files failed CheckParity:\n- %s", strings.Join(v, "\n- "))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestAdopterRulesParityDisagreement(t *testing.T) {
	rows := []int{2, 1, 3, 1, 2}
	agents := adopterRules("### ", rows)
	cases := []struct {
		name  string
		index string
		want  []string
	}{
		{"title differs", strings.Replace(adopterRules("## ", rows), "3. Release train", "3. Release trains", 1), []string{"rule section 3", "AGENTS.md", ".agents/rules-index.md", "3. Release train"}},
		{"row count differs", adopterRules("## ", []int{2, 1, 4, 1, 2}), []string{"row count", "3. Release train", "AGENTS.md has 3", ".agents/rules-index.md has 4"}},
		{"section missing from index", strings.Replace(adopterRules("## ", rows), "5. On-call", "On-call", 1), []string{"5. On-call", "missing from .agents/rules-index.md"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := brownfield(t, agents, c.index, adopterProtocol)
			requireViolation(t, CheckParity(dir), c.want...)
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Adopter rules parity
func TestAdopterProtocolMarkers(t *testing.T) {
	rows := []int{1, 1, 1, 1, 1}
	agents, index := adopterRules("### ", rows), adopterRules("## ", rows)
	for _, c := range []struct{ marker, want string }{
		{"Deliverable", "Clarify step"},
		{"Scope", "Clarify step"},
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
		requireViolation(t, v, `"Deliverable"`, `"Scope"`, `"Definition of Done"`, `"Stop boundary"`)
	})
}
