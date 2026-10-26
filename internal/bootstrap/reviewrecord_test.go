package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const recordTiers = `"model_tiers":{"frontier":["claude:opus"],"workhorse":["claude:sonnet"]},"effort_bands":{"claude":{"high":["high"]},"codex":{"high":["high"]},"gemini":{"high":["high"]}}`

// initWithPolicy runs ateam init in a fresh directory holding policy ("" for none)
// and returns every generated template file as written.
func initWithPolicy(t *testing.T, policy string) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	if policy != "" {
		if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, reviewPolicyPath), []byte(policy), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatalf("init: %v", err)
	}
	got := map[string]string{}
	for rel := range defaultTemplateFiles() {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		got[rel] = string(data)
	}
	return root, got
}

// The PR-comment instruction as it read before #130; the default must keep it.
const prCommentInstruction = "in a PR comment starting `reviewed-by: <agent> (<model>, <effort>)` and naming one PR commit as `Reviewed at <sha>`"

// spec: openspec/specs/model-review/spec.md#Requirement: Review record mode
func TestInitReviewRecordDefaultUnchanged(t *testing.T) {
	_, none := initWithPolicy(t, "")
	if !strings.Contains(none[".agents/protocol.md"], prCommentInstruction) {
		t.Fatal("default protocol lost the PR-comment record instruction")
	}
	for name, policy := range map[string]string{
		"absent key": `{"version":1,` + recordTiers + `}`,
		"pr-comment": `{"version":1,"review_record":"pr-comment",` + recordTiers + `}`,
	} {
		_, got := initWithPolicy(t, policy)
		for rel, want := range none {
			if got[rel] != want {
				t.Errorf("%s: %s differs from the output with no policy", name, rel)
			}
		}
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Review record mode
func TestInitReviewRecordAIP(t *testing.T) {
	_, none := initWithPolicy(t, "")
	root, aip := initWithPolicy(t, `{"version":1,"review_record":"aip",`+recordTiers+`}`)
	changed := map[string]bool{"AGENTS.md": true, ".agents/rules-index.md": true, ".agents/protocol.md": true, ".agents/contracts/reviewer.md": true, ".agents/skills/review-pr/SKILL.md": true}
	for rel, text := range aip {
		if !changed[rel] {
			if text != none[rel] {
				t.Errorf("%s changed under aip", rel)
			}
			continue
		}
		for _, banned := range []string{"Reviewed at", "PR comment review record", "Post the record as one pull request comment", "gh pr comment"} {
			if strings.Contains(text, banned) {
				t.Errorf("%s under aip still says %q", rel, banned)
			}
		}
		if strings.HasSuffix(rel, "SKILL.md") {
			continue
		}
		for _, want := range []string{"openspec/changes/archive/<date>-<id>/review-implementation.md", "`Verdict: PASS`", "`approval.md`", "never post a `reviewed-by:`", "`task merge:check`", "`--ff-only`"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s under aip lacks %q", rel, want)
			}
		}
	}
	if !strings.Contains(aip[".agents/contracts/reviewer.md"], "--reviewer-record") {
		t.Error("aip reviewer contract does not name ateam review compare --reviewer-record")
	}
	// Doctor accepts the aip output: rule row counts match and the contract is the aip one.
	if v := CheckParity(root); len(v) != 0 {
		t.Fatalf("aip init fails parity: %v", v)
	}
	// The default contract in an aip repository is drift.
	if err := os.WriteFile(filepath.Join(root, ".agents/contracts/reviewer.md"), []byte(none[".agents/contracts/reviewer.md"]), 0o644); err != nil {
		t.Fatal(err)
	}
	requireViolation(t, checkReviewSetup(root), ".agents/contracts/reviewer.md")
}

// spec: openspec/specs/model-review/spec.md#Requirement: Review record mode
func TestInitReviewRecordInvalidPolicy(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, reviewPolicyPath), []byte(`{"version":1,"review_record":"file",`+recordTiers+`}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := InitRepo(context.Background(), root, Options{})
	if err == nil || !strings.Contains(err.Error(), "review_record") || len(res.Written) != 0 {
		t.Fatalf("init with an unknown review_record: written %v, err %v", res.Written, err)
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Review record mode
func TestConfigureReviewRecordAIP(t *testing.T) {
	root := t.TempDir()
	if _, err := ConfigureReview(root, []byte(`{"version":1,"review_record":"aip",`+recordTiers+`}`)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".agents/contracts/reviewer.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "never post a `reviewed-by:`") || strings.Contains(string(data), "Reviewed at") {
		t.Fatal("review setup under aip wrote the PR-comment contract")
	}
	if v := checkReviewSetup(root); len(v) != 0 {
		t.Fatalf("aip setup drifts: %v", v)
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Review record mode
func TestInitReviewRecordAIPHasNoPRWorkflow(t *testing.T) {
	_, none := initWithPolicy(t, "")
	root, aip := initWithPolicy(t, `{"version":1,"review_record":"aip",`+recordTiers+`}`)
	// PR and squash-merge commands that an AIP branch, which merges --ff-only
	// with no PR, must not be told to run.
	prWorkflow := []string{"draft PR", "gh pr ", "--squash", "squash auto-merge", "merge_when_checks_succeed", "/pulls/", "marked ready", "Mark the PR ready", "reviewed-by: <agent> (<model>, <effort>)"}
	for _, rel := range []string{".agents/protocol.md", "AGENTS.md", ".agents/rules-index.md", ".agents/skills/review-pr/SKILL.md"} {
		hits := 0
		for _, cmd := range prWorkflow {
			if strings.Contains(none[rel], cmd) {
				hits++
			}
			if strings.Contains(aip[rel], cmd) {
				t.Errorf("%s under aip still says %q", rel, cmd)
			}
		}
		if hits == 0 {
			t.Errorf("%s: the default names none of the PR commands, so this check proves nothing", rel)
		}
	}
	for _, want := range []string{"open no PR", "`--ff-only` merge into `main`", "remove the worktree you claimed"} {
		if !strings.Contains(aip[".agents/protocol.md"], want) {
			t.Errorf("aip protocol lacks %q", want)
		}
	}
	// Doctor accepts the aip review-pr skill, and still wants one full marker set.
	skill := filepath.Join(root, ".agents/skills/review-pr/SKILL.md")
	if err := os.WriteFile(skill, []byte(strings.ReplaceAll(aip[".agents/skills/review-pr/SKILL.md"], "--reviewer-record", "--reviewer")), 0o644); err != nil {
		t.Fatal(err)
	}
	requireViolation(t, CheckParity(root), ".agents/skills/review-pr/SKILL.md")
}
