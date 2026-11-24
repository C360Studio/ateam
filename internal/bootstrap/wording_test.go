package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adopterWording pins the four rules from an adopter's review (#400). The
// expected sentences are literals copied from the approved proposal's wording,
// not derived from the templates.
var adopterWording = map[string][]string{
	".agents/skills/ponytail/SKILL.md": {
		"Name the caller or spec scenario that needs it today; if there is none, do not build it.",
		"Defensive code names the concrete failure it prevents and how that failure occurs.",
		"A defence that turns a transient fault into a safety action is itself a hazard and needs the same justification as a feature.",
	},
	".agents/protocol.md": {
		"expected values are literals, a hand derivation shown in the test, or data from a named external source",
		"A helper that mirrors the control flow of the code under test is not an oracle",
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
