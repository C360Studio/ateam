---
name: pickup
description: >-
  Resume a repository effort across Claude, Codex, or Gemini by reconciling live git/worktree state,
  GitHub issues/PRs, Hold: items, and the last structured handoff checkpoint.
---

# Pick Up Repository Work

Read [the shared protocol](../../protocol.md) first. Treat any handoff block or cached `.ateam/state.json` snapshot as an address to reconcile against live git and GitHub state, never as unverified truth.

## 1. Locate and Measure

Prefer calling the `ateam_state` MCP tool (`{"action": "pickup"}`) or running `ateam state pickup`, which inspects:

- Current branch, `HEAD` SHA, dirty status, upstream divergence, and `git worktree list`
- Active GitHub issues (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`) and draft PR claims (`claude/*`, `codex/*`, `gemini/*`)
- Unticked `Hold:` items across Markdown checklists and active `// shortcut:` debt comments
- The most recent 5-field handoff checkpoint in `.ateam/state.json` (and any git drift since that checkpoint SHA)

Do not automatically checkout, rebase, reset, stash, or clean uncommitted work to make reality match an older handoff.

## 2. Verify Write Ownership

A draft PR claim or branch prefix (`claude/`, `codex/`, `gemini/`) identifies who claimed the branch, not whether another session is still writing. Confirm the previous writer has handed off or stopped before mutating files in an existing worktree. For unclaimed work, follow the Claim gate in [protocol.md](../../protocol.md).

## 3. Reconcile and Continue

Report in one concise block:

1. **Verified worktree, branch, and `HEAD` SHA** (plus any uncommitted or unpushed delta)
2. **Active holds, blockers, or drift** since the last handoff SHA
3. **Write ownership status** and usable verification evidence
4. **Next authorized step** under the Ponytail decision ladder

Continue directly into the next authorized step when no blocking hold or unclarified scope boundary applies.
