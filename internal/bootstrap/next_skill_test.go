package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec: harness / Shared computed next issue
func TestPickupAndHandoffTakeNextFromAteam(t *testing.T) {
	// Both the generated skills and this repository's copies must send "next" to the command, so no session carries
	// its own order in a handoff (#418).
	repoRoot := filepath.Join("..", "..")
	for _, name := range []string{"pickup", "handoff"} {
		repoCopy, err := os.ReadFile(filepath.Join(repoRoot, ".agents", "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		for where, text := range map[string]string{"template": canonicalSkillBodies[name], "repository": string(repoCopy)} {
			if !strings.Contains(text, "`ateam state next`") {
				t.Errorf("%s %s skill does not name `ateam state next`", where, name)
			}
		}
	}
	if !strings.Contains(canonicalSkillBodies["handoff"], "milestone order lives in the forge") {
		t.Error("handoff template Goal still invites an order")
	}
}
