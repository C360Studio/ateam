# Change Proposal: `worktree-handoffs-136`

## Why

Each linked worktree resolves its own repository root, so a handoff recorded inside a worktree lands in that worktree's `.ateam/state.json`. Pickup from `main` listed the worktrees but none of their handoffs, so a fan-out of ten worktree handoffs read as `Last Handoff: none` (#136).

## What Changes

- Pickup reads each other worktree's `.ateam/state.json` (read-only) and prints its handoff for that branch under the worktree line, marked `stale` when it is. Where handoffs are stored does not change.
