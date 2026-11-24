package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adopterWording pins the rules from an adopter's review (#400, #398). The
// expected sentences are literals copied from the approved proposal's wording,
// not derived from the templates.
var adopterWording = map[string][]string{
	"AGENTS.md": {
		"Name the caller or spec scenario that needs it today; if there is none, do not build it.",
		"Defensive code names the concrete failure it prevents and how that failure occurs.",
	},
	".agents/skills/ponytail/SKILL.md": {
		"Name the caller or spec scenario that needs it today; if there is none, do not build it.",
		"Defensive code names the concrete failure it prevents and how that failure occurs.",
		"A defence that turns a transient fault into a safety action is itself a hazard and needs the same justification as a feature.",
	},
	".agents/protocol.md": {
		"section whose bullets say in plain words what each decision triggers and changes for the user",
		"expected values are literals, a hand derivation shown in the test, or data from a named external source",
		"A helper that mirrors the control flow of the code under test is not an oracle",
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
