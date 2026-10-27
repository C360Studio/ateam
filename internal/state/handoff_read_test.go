package state

import (
	"strings"
	"testing"
	"time"
)

func clearAgentSessionEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ATEAM_AGENT", "ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CLAUDECODE", "CODEX_SESSION_ID", "GEMINI_SESSION_ID"} {
		t.Setenv(k, "")
	}
}

// spec: openspec/changes/handoff-read-196/specs/harness/spec.md#Requirement: Read another agent's handoff without changing ownership
func TestPickupPreviewsOtherHandoffGoal(t *testing.T) {
	// #196: a Codex pickup must show what the Claude slot is about, attributed to Claude.
	clearAgentSessionEnv(t)
	longGoal := "Milestone v0.3.3: land #198 then #196 then #193, keep slot isolation, never touch gate files, run task verify before ready"
	for _, tc := range []struct {
		name, goal string
		legacy     bool
		want       string
	}{
		{
			name: "slot map, goal clipped at 104 bytes on a word boundary",
			goal: longGoal,
			want: "Last Handoff: none for codex@main\nother handoffs: claude@main (5m ago): Milestone v0.3.3: land #198 then #196 then #193, keep slot isolation, never touch gate files, run task…",
		},
		{
			name:   "legacy last_handoff",
			goal:   "Ship the read path",
			legacy: true,
			want:   "Last Handoff: none for codex@main\nother handoffs: claude@main (5m ago): Ship the read path",
		},
		{
			name: "empty goal adds no suffix",
			goal: "  ",
			want: "Last Handoff: none for codex@main\nother handoffs: claude@main (5m ago)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := defaultState()
			st.Git.Branch, st.Git.HeadSHA = "main", "1111111"
			h := Handoff{Goal: tc.goal, Agent: "claude", Branch: "main", CheckpointSHA: "1111111", UpdatedAt: time.Now().UTC().Add(-5 * time.Minute)}
			if tc.legacy {
				st.LastHandoff = &h
			} else {
				st.SetHandoff(h)
			}
			st.CurrentAgent = "codex"
			summary := BuildPickupSummary(st)
			if !strings.HasSuffix(summary, tc.want) {
				t.Fatalf("pickup handoff section mismatch:\ngot:\n%s\nwant suffix:\n%s", summary, tc.want)
			}
		})
	}
}

// spec: openspec/changes/handoff-read-196/specs/harness/spec.md#Requirement: Read another agent's handoff without changing ownership
func TestReadHandoffSlots(t *testing.T) {
	clearAgentSessionEnv(t)
	ago := time.Now().UTC().Add(-5 * time.Minute)
	slot := func(agent, branch, sha, goal string) Handoff {
		return Handoff{Goal: goal, Addresses: "issue #196", ChecksToRun: "task verify", Agent: agent, Branch: branch, CheckpointSHA: sha, UpdatedAt: ago}
	}
	save := func(dir, branch, sha string, hs ...Handoff) {
		st := defaultState()
		st.Git.Branch, st.Git.HeadSHA = branch, sha
		for _, h := range hs {
			st.SetHandoff(h)
		}
		if err := Save(dir, st); err != nil {
			t.Fatal(err)
		}
	}
	root, feat, stale, dup := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	save(feat, "claude/feat", "aaaaaaa", slot("claude", "claude/feat", "aaaaaaa", "worktree goal"))
	save(stale, "claude/stale", "bbbbbbb", slot("claude", "claude/stale", "bbbbbbb", "stale goal"))
	save(dup, "codex/dup", "ddddddd", slot("codex", "codex/dup", "ddddddd", "worktree copy"))
	rootSt := defaultState()
	rootSt.Git.Branch, rootSt.Git.HeadSHA = "main", "1111111"
	rootSt.Git.Worktrees = []Worktree{
		{Path: root, Branch: "main", Head: "1111111"},
		{Path: feat, Branch: "claude/feat", Head: "aaaaaaa"},
		{Path: stale, Branch: "claude/stale", Head: "ccccccc"}, // moved past its checkpoint
		{Path: dup, Branch: "codex/dup", Head: "ddddddd"},
	}
	rootSt.SetHandoff(slot("claude", "main", "1111111", "current checkout goal"))
	rootSt.SetHandoff(slot("codex", "codex/dup", "ddddddd", "current copy"))
	if err := Save(root, rootSt); err != nil {
		t.Fatal(err)
	}

	fields := func(goal string) string {
		return "Goal: " + goal + "\nAddresses: issue #196\nConstraints: none\nDone when: none\nChecks to run: task verify\n"
	}
	for _, tc := range []struct {
		name, slot, want, wantErr string
	}{
		{"current checkout", "claude@main", "Handoff claude@main (5m ago):\n" + fields("current checkout goal"), ""},
		{"worktree only", "claude@claude/feat", "Handoff claude@claude/feat (5m ago):\n" + fields("worktree goal"), ""},
		{"duplicate, current wins", "codex@codex/dup", "Handoff codex@codex/dup (5m ago):\n" + fields("current copy"), ""},
		{"stale under banner", "claude@claude/stale", "STALE — HEAD drifted from bbbbbbb -> ccccccc\nHandoff claude@claude/stale (5m ago):\n" + fields("stale goal"), ""},
		{"missing lists slots", "codex@main", "", "no handoff for codex@main (pickup prunes stale slots); existing slots: claude@claude/feat, claude@claude/stale, claude@main, codex@codex/dup"},
		{"malformed slot", "claude", "", `slot "claude": want <agent>@<branch>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadHandoff(root, tc.slot)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("ReadHandoff(%q) error = %v, want %q", tc.slot, err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ReadHandoff(%q) mismatch (err %v):\ngot:\n%s\nwant:\n%s", tc.slot, err, got, tc.want)
			}
		})
	}
}
