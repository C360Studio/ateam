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
			// status-label-route-443 D1-D4.
			"put your answer on a line starting `Ruling:` in your comment, and remove that label",
			"every issue its `Depends on:` lines name is closed and every pull request they name is merged",
			"if any issue is open, or any pull request is open or closed without merging, answer \"cannot decide\"",
			"Never add a label, and never remove one you were not asked about: `gate-change-approved` and `class:flake` stay the owner's.",
		},
		".agents/contracts/coordinator.md": {
			"A change touching a gate file and a release still stop for the owner.",
			"an issue labelled `status:needs-decision`, and an issue labelled `status:blocked` that `ateam state queue-check` reports with every `Depends on:` issue closed.",
			"Any other `status:blocked` issue stays an owner gate and does not end the run.",
			"run `ateam review compare --implementer <agent:model:effort>` for each run it rules on with `--reviewer <the judge run>`",
			"End the run and hand back to the owner when a judge answers \"cannot decide\", `main` CI goes red, a unit touches a gate file, or the milestone you started on is finished.",
		},
		".agents/skills/coordinate/SKILL.md": {
			"the developer run fixes the findings it rules blocking and files the rest as follow-up issues, then request a fresh review",
			"or a status-labelled issue's number with its `ateam state queue-check` finding)",
			"After a judge removes a status label, run `ateam state sync`, so `ateam state next` can name the issue.",
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

// spec: model-review / Owner decision after the third rejecting review
func TestInitShipsRoundsContinueRule(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	// Literals from the approved proposal's D2: who writes the continue line, and who never does.
	for rel, want := range map[string][]string{
		".agents/contracts/coordinator.md": {"review records, `Approved-by:` or a `review-rounds-continue:` line;"},
		".agents/contracts/judge.md":       {"end that comment with the line `review-rounds-continue: <n>` outside code fences"},
		".agents/skills/coordinate/SKILL.md": {
			"`ateam review request` refuses the next round until the owner posts a `review-rounds-continue: <n>` line naming that count",
			"never post the line yourself",
			"which the judge's `review-rounds-continue: <n>` line allows",
		},
	} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range want {
			if !strings.Contains(string(data), s) {
				t.Errorf("shipped %s lacks %q", rel, s)
			}
		}
	}
}

// spec: harness / Optional coordinator role
func TestInitShipsReviewOnGreenChecks(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	// Literals from the approved proposal review-on-green-checks-446: a ready
	// handoff needs green required checks, and the coordinator reads them first.
	for rel, want := range map[string][]string{
		".agents/contracts/developer.md": {
			"Hand off at ready-for-review (the reason `ready`) only once every required check at the pull request head has passed, apart from `Merge check`",
			"when one fails, fix it and push, or hand off as `blocked` naming the check.",
		},
		".agents/skills/coordinate/SKILL.md": {
			"When a developer hands off at ready-for-review, first read the required checks at the PR head (`gh pr checks <pr> --required`;",
			"has not reported yet (the command lists only checks that have reported), message the developer run with that check and its state instead of requesting review. Otherwise run `ateam review request --return-to <your id> <pr>`",
		},
	} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range want {
			if !strings.Contains(string(data), s) {
				t.Errorf("shipped %s lacks %q", rel, s)
			}
		}
	}
}

// spec: harness / Optional coordinator role
func TestInitShipsDeveloperHandoffFields(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	// Literals from the approved proposal developer-handoff-fields-447: what a
	// developer handoff and PR body must surface, and what the owner message repeats.
	for rel, want := range map[string][]string{
		".agents/contracts/developer.md": {
			"An action only the owner can take goes in the handoff Goal, named as owed by the owner",
			"The handoff names the change's track and, for `quick`, the one-line reason no design trigger applies.",
			"never write \"no gate file touched\"; quote the line `scripts/gate-files.sh <pr>` printed and the commit it ran at.",
			"When you rewrite your pull request body, keep any `owner-prompts:` line you find, unchanged.",
		},
		".agents/skills/coordinate/SKILL.md": {
			"With each proposal, repeat the track its developer handoff names and, for `quick`, its reason, and list every action a developer handoff Goal names as owed by the owner.",
		},
	} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range want {
			if !strings.Contains(string(data), s) {
				t.Errorf("shipped %s lacks %q", rel, s)
			}
		}
	}
}

// spec: harness / Optional coordinator role
func TestInitShipsOwnerMessageRoutes(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	// Literals from the approved proposal owner-message-routes-449: the owner
	// message links the developer run first, offers three approval routes and
	// drops a refused one, reads before asking, and counts a repeated gate once.
	for rel, want := range map[string][]string{
		".agents/skills/coordinate/SKILL.md": {
			"Before each owner message, run `ateam state sync` and, for every gate you would list, read its branch (`Approved-by:` filled, the hold gone) and the PR's comments since you last asked; drop a gate already answered",
			"the developer run's conversation link when it has one (the one your launch tool returned), then the PR or issue,",
			"For a proposal, offer three ways to approve: answer you, and you pass the owner's words unchanged to that developer run; say it in the developer's conversation; or commit the `Approved-by:` edit themselves.",
			"Leave out the first when that developer run handed off as `blocked` because its host refused the edit.",
			"each PR that message newly asks about",
			"a gate already asked and still unanswered adds nothing when you repeat it.",
		},
		".agents/contracts/coordinator.md": {
			"each pull request that message newly asks about, never for a gate repeated while still unanswered;",
		},
		".agents/contracts/developer.md": {
			"Write `Approved-by:` only on the owner's own words, given in your conversation or passed on unchanged by the coordinator;",
			"a summary of the answer, or an approval naming a digest other than the one `task spec:queue` prints, is not one, so ask again.",
			"quote the owner's words and where they came from in the commit message.",
			"When your host refuses the edit, do not retry it: hand off as `blocked` naming the refusal,",
		},
	} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range want {
			if !strings.Contains(string(data), s) {
				t.Errorf("shipped %s lacks %q", rel, s)
			}
		}
	}
	// The old wording counted every message that covered a pull request.
	contract, err := os.ReadFile(filepath.Join(dir, ".agents", "contracts", "coordinator.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contract), "that message covers;") {
		t.Error("shipped coordinator contract still counts every message that covers a pull request")
	}
}
