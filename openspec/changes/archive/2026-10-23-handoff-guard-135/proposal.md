# Change Proposal: `handoff-guard-135`

## Why

The handoff slot is keyed `agent@branch`. When several sessions of one agent hand off on the same branch, every write replaces the last with no signal: in a ten-session fan-out, one handoff survived and all ten exited 0 (#135). #124 fixed the same loss on the pre-compact path; the deliberate write paths (CLI, MCP, HTTP) have no guard.

## What Changes

- A deliberate handoff into a slot holding a fresh handoff from a different or unknown session is refused, naming the holder; nothing is written or posted. The predicate is the one pre-compact uses (#124): not stale, not an auto-save, session differs or is unknown (owner ruling on #135: an unknown session is protected).
- `--force` (CLI), `force: true` (MCP), `?force=1` (HTTP) replace the slot anyway.

## Non-goals

- Keying slots by session, and showing worktree handoffs from `main` (#136).
