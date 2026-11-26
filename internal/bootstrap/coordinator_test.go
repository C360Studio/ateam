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

// spec: harness / Judge approvals under the hold-my-beer opt-in
func TestInitShipsJudgeContract(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	read := func(root, rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	// The templates are the files this repository dogfoods, not paraphrases of them.
	for _, rel := range []string{".agents/contracts/judge.md", ".agents/contracts/coordinator.md", ".agents/skills/coordinate/SKILL.md"} {
		if read(dir, rel) != read(repoRoot(t), rel) {
			t.Fatalf("shipped %s differs from the repository file", rel)
		}
	}
	// Literals from the approved proposal's decisions (hold-my-beer-424, D2-D6 and ruling 5).
	for rel, want := range map[string][]string{
		".agents/contracts/judge.md": {
			"while `.agents/review-policy.json` sets `\"hold_my_beer\": true`",
			"in at most 20 tool calls",
			"\"Cannot decide\" is a valid answer",
			"`Judged-by: <agent> (<model>, <effort>) sha256:<12 hex>` on its own line below `Approved-by:`",
			"Never write or edit `Approved-by:`",
			"Never write a review record or turn a rejection into an approval",
			"Never apply `gate-change-approved`, tag or release",
			"comment on each issue you move with the old and new line",
		},
		".agents/contracts/coordinator.md": {
			"A change touching a gate file and a release still stop for the owner.",
			"run `ateam review compare --implementer <agent:model:effort>` for each run it rules on with `--reviewer <the judge run>`",
			"End the run and hand back to the owner when a judge answers \"cannot decide\", `main` CI goes red, a unit touches a gate file, or the milestone you started on is finished.",
		},
		".agents/skills/coordinate/SKILL.md": {
			"the developer run fixes the findings it rules blocking and files the rest as follow-up issues, then request a fresh review",
		},
	} {
		body := read(dir, rel)
		for _, s := range want {
			if !strings.Contains(body, s) {
				t.Errorf("%s lacks %q", rel, s)
			}
		}
	}
	if v := CheckParity(dir); len(v) != 0 {
		t.Fatalf("fresh init has violations: %v", v)
	}
}
