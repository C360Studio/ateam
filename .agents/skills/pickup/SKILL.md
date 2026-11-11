---
name: pickup
description: >-
  Resume a repository effort across Claude, Codex, or Gemini by reconciling live git/worktree state,
  GitHub issues/PRs, Hold: items, and the last structured handoff checkpoint.
---

# Pick Up Repository Work

From [the shared protocol](../../protocol.md), read § 1 and § 2 steps 1–2 (Clarify, Claim); an unattended or fan-out run also reads § 4 *Unattended and fan-out runs*. If your protocol has no such numbered sections, read all of it. Treat any handoff block or cached `.ateam/state.json` snapshot as an address to reconcile against live git and GitHub state, never as unverified truth.

## 1. Locate and Measure

Prefer calling the `ateam_state` MCP tool (`{"action": "pickup"}`) or running `ateam state pickup`, which inspects:

- Current branch, `HEAD` SHA, dirty status, upstream divergence, and `git worktree list`
- Active GitHub issues (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`) and draft PR claims (`claude/*`, `codex/*`, `gemini/*`)
- Unticked `Hold:` items across Markdown checklists and active `// shortcut:` debt comments
- The most recent 5-field handoff checkpoint in `.ateam/state.json` (and any git drift since that checkpoint SHA)

Do not automatically checkout, rebase, reset, stash, or clean uncommitted work to make reality match an older handoff.

## 2. Verify Write Ownership

A draft PR claim or branch prefix (`claude/`, `codex/`, `gemini/`) identifies who claimed the branch, not whether another session is still writing. Confirm the previous writer has handed off or stopped before mutating files in an existing worktree. For unclaimed work, follow the Claim gate in [protocol.md](../../protocol.md).

Inside herdr (`HERDR_ENV=1`), the writer lease and the different-writer warning name the herdr pane (`herdr pane <id>`); check that pane before assuming the writer stopped. Claim new work with `herdr worktree create --cwd "$PWD" --branch <agent>/<topic> --path "$PWD/.worktrees/<agent>-<topic>" --no-focus`; `git worktree list` stays the source of truth.

## 3. Reconcile and Continue

Report in one concise block:

1. **Verified worktree, branch, and `HEAD` SHA** (plus any uncommitted or unpushed delta)
2. **Active holds, blockers, or drift** since the last handoff SHA
3. **Write ownership status** and usable verification evidence
4. **Next authorized step** under the Ponytail decision ladder

Continue directly into the next authorized step when no blocking hold or unclarified scope boundary applies.

**T3 Code card.** Only when the `html_render` tool is available to you and `printenv T3CODE_THREAD_ID` prints a non-empty value: after the report above, read [t3-card.html](../handoff/t3-card.html), replace each double-brace placeholder with a fact read this turn (escape `& < > " '` first, write `not checked` for anything you did not read, and put the handoff goal and done-when only in the Agent-written region), then call `html_render` with that HTML, the title `Pickup · <branch>`, and height 560. On any other host, skip this step.
