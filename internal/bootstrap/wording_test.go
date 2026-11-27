package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adopterWording pins the rules from an adopter's review (#400, #398) and the
// Close rule's archive after approval (#412). The
// expected sentences are literals copied from the approved proposal's wording,
// not derived from the templates.
const closeArchiveRow = "A PR archives its completed OpenSpec change in a document-only commit after its approving review record and before auto-merge is armed (on Gitea, before it is marked ready);"

var adopterWording = map[string][]string{
	"AGENTS.md": {
		"Name the caller or spec scenario that needs it today; if there is none, do not build it.",
		"Defensive code names the concrete failure it prevents and how that failure occurs.",
		closeArchiveRow,
	},
	".agents/rules-index.md": {
		closeArchiveRow,
	},
	".agents/skills/ponytail/SKILL.md": {
		"Name the caller or spec scenario that needs it today; if there is none, do not build it.",
		"Defensive code names the concrete failure it prevents and how that failure occurs.",
		"A defence that turns a transient fault into a safety action is itself a hazard and needs the same justification as a feature.",
	},
	".agents/protocol.md": {
		"When an epic or milestone closes, a whole-repository review is due",
		"section whose bullets say in plain words what each decision triggers and changes for the user",
		"expected values are literals, a hand derivation shown in the test, or data from a named external source",
		"A helper that mirrors the control flow of the code under test is not an oracle",
		"Once the latest review record approves, move the change to `openspec/changes/archive/<date>-<id>/` in one commit touching only documents, which on GitHub needs no new record, and only then arm auto-merge.",
		"On Gitea, where every commit after the record needs a new one, make that move before the PR is marked ready.",
	},
	".agents/contracts/architect.md": {
		"Keep the repository read-only except the change's `design.md`, the premises in its `tasks.md` and the bullets under `## Decisions you are making by approving` in its `proposal.md`;",
		"List each decision the design asks the owner to make in `proposal.md` under `## Decisions you are making by approving`",
	},
	".agents/skills/design/SKILL.md": {
		"`task spec:check` fails a change with a `design.md` and no such bullet.",
	},
	"openspec/config.yaml": {
		"Open with a '## Decisions you are making by approving' section",
	},
	".agents/contracts/reviewer.md": {
		"For each test oracle, ask where its expected values came from",
		"A claim in a title, spec, comment or record cites its artifact",
		"reject `verified` or `proves` without an artifact.",
	},
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestAdopterWordingInGeneratedAndRepoFiles(t *testing.T) {
	generated := t.TempDir()
	if _, err := InitRepo(context.Background(), generated, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	for root, label := range map[string]string{generated: "generated", filepath.Join("..", ".."): "repository"} {
		for rel, sentences := range adopterWording {
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("%s %s: %v", label, rel, err)
			}
			for _, s := range sentences {
				if !strings.Contains(string(body), s) {
					t.Errorf("%s %s lacks %q", label, rel, s)
				}
			}
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Whole-repository review marker
func TestWholeReviewAdviceUnderAIP(t *testing.T) {
	// The AIP Close step replaces the default one, so it must carry the #399 advice on its own.
	_, aip := initWithPolicy(t, `{"version":1,"review_record":"aip",`+recordTiers+`}`)
	if want := "When an epic or milestone closes, a whole-repository review is due"; !strings.Contains(aip[".agents/protocol.md"], want) {
		t.Fatalf("AIP protocol lacks %q", want)
	}
}
