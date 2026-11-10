package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestScanWorktreeChanges(t *testing.T) {
	tmp, err := filepath.EvalSymlinks(t.TempDir()) // git prints resolved paths (/private/var on macOS)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(tmp, "root")
	gitIn(t, tmp, "init", "-q", "-b", "main", root)
	gitIn(t, root, "commit", "-q", "--allow-empty", "-m", "init")

	// The change awaiting approval lives only on the agent's branch, never in root.
	held := filepath.Join(tmp, "held")
	gitIn(t, root, "worktree", "add", "-q", "-b", "claude/held", held)
	const heldProposal = "# Change Proposal: `alpha`\n\nApproved-by:\n\n## Why\n\nBecause.\n"
	writeRepoFile(t, held, "openspec/changes/alpha/proposal.md", heldProposal)
	writeRepoFile(t, held, "openspec/changes/alpha/tasks.md", "## 1. Approval\n\n- [ ] 1.1 Owner approves. Hold: OPEN-Q proposal awaits owner approval\n")

	// A digest of zeros cannot match any proposal, so the state read from disk is stale.
	stale := filepath.Join(tmp, "stale")
	gitIn(t, root, "worktree", "add", "-q", "-b", "claude/stale", stale)
	writeRepoFile(t, stale, "openspec/changes/beta/proposal.md", "# Beta\n\nApproved-by: Ann Owner (2026-11-07) sha256:000000000000\n")
	writeRepoFile(t, stale, "openspec/changes/beta/tasks.md", "## 1. Work\n\n- [x] 1.1 Done.\n")

	// A detached review checkout of the held branch must not list alpha a second time.
	gitIn(t, held, "add", "-A")
	gitIn(t, held, "commit", "-q", "-m", "alpha")
	review := filepath.Join(tmp, "review")
	gitIn(t, root, "worktree", "add", "-q", "--detach", review, "claude/held")

	// tasks.md as a directory makes the scan fail in this worktree only.
	broken := filepath.Join(tmp, "broken")
	gitIn(t, root, "worktree", "add", "-q", "-b", "claude/broken", broken)
	if err := os.MkdirAll(filepath.Join(broken, "openspec", "changes", "gamma", "tasks.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	changes, skipped := ScanWorktreeChanges(context.Background(), root)

	type row struct{ name, branch, state, proposal string }
	var got []row
	for _, ch := range changes {
		got = append(got, row{ch.Name, ch.Branch, ch.Approval.State, ch.Proposal})
	}
	want := []row{
		{"alpha", "claude/held", ApprovalUnapproved, heldProposal},
		{"beta", "claude/stale", ApprovalStale, ""},
	}
	if len(got) != len(want) {
		t.Fatalf("changes = %+v, want %d rows %+v", got, len(want), want)
	}
	for i := range want {
		w := want[i]
		if w.proposal == "" {
			got[i].proposal = "" // only alpha's text is asserted byte for byte
		}
		if got[i] != w {
			t.Errorf("change %d = %+v, want %+v", i, got[i], w)
		}
	}
	if !hasCaveat(changes[0].Caveats, "OPEN-Q") {
		t.Errorf("alpha caveats = %+v, want an OPEN-Q", changes[0].Caveats)
	}
	if changes[0].Worktree != held {
		t.Errorf("alpha worktree = %q, want %q", changes[0].Worktree, held)
	}
	if len(skipped) != 1 || !strings.HasPrefix(skipped[0], broken+": ") {
		t.Errorf("skipped = %q, want one entry for %s", skipped, broken)
	}
}

func hasCaveat(cs []OpenSpecCaveat, marker string) bool {
	for _, c := range cs {
		if c.Marker == marker {
			return true
		}
	}
	return false
}
