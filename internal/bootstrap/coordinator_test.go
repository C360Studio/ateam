package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Optional coordinator role
// spec: openspec/specs/harness/spec.md#Requirement: Adopter skills and adapter aliases
func TestInitShipsCoordinatorAsCanonical(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	files := []string{
		".agents/contracts/coordinator.md",
		".agents/skills/coordinate/SKILL.md",
		".claude/skills/coordinate/SKILL.md",
		".gemini/commands/coordinate.toml",
	}
	for _, rel := range files {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("InitRepo did not write %s: %v", rel, err)
		}
	}
	// The templates are the files this repository dogfoods, not paraphrases of them.
	for _, rel := range files[:2] {
		shipped, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		repo, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if string(shipped) != string(repo) {
			t.Fatalf("shipped %s differs from the repository file", rel)
		}
	}
	// Literals from the approved proposal's decisions (coordinator-417, D3 and the stop list).
	contract, _ := os.ReadFile(filepath.Join(dir, ".agents", "contracts", "coordinator.md"))
	for _, s := range []string{
		"never apply `gate-change-approved`",
		"never change labels, milestones or an `Order:` line",
		"a third `Verdict: REQUEST CHANGES` on one pull request",
		"`.worktrees/<agent>-coordinator` on `<agent>/coordinator`",
	} {
		if !strings.Contains(string(contract), s) {
			t.Fatalf("coordinator contract lacks %q", s)
		}
	}
	if v := CheckParity(dir); len(v) != 0 {
		t.Fatalf("fresh init has violations: %v", v)
	}

	// doctor treats coordinate as canonical: dropping a marker or the generic file is a violation,
	// and the skill is never listed as an adopter skill.
	p := filepath.Join(dir, ".agents", "skills", "coordinate", "SKILL.md")
	body, _ := os.ReadFile(p)
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(string(body), "owner-prompts:", "count")), 0o644); err != nil {
		t.Fatal(err)
	}
	requireViolation(t, CheckParity(dir), ".agents/skills/coordinate/SKILL.md", "owner-prompts:")
	if adopter, _ := SkillInventory(dir); len(adopter) != 0 {
		t.Fatalf("coordinate listed as adopter skill: %v", adopter)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	requireViolation(t, CheckParity(dir), ".agents/skills/coordinate/SKILL.md: missing canonical skill")
}
