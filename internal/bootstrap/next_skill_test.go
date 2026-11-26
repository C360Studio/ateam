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

// spec: harness / Queue health checks
func TestPreflightNamesQueueCheck(t *testing.T) {
	// The owner asked for the queue findings at preflight (#418 comment, 2026-11-26), so both copies name the command.
	repoCopy, err := os.ReadFile(filepath.Join("..", "..", ".agents", "skills", "preflight", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for where, text := range map[string]string{"template": canonicalSkillBodies["preflight"], "repository": string(repoCopy)} {
		if !strings.Contains(text, "`ateam state queue-check`") || !strings.Contains(text, "handoff") {
			t.Errorf("%s preflight skill does not name `ateam state queue-check` before a handoff", where)
		}
	}
}
