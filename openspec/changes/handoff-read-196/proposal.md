# Change Proposal: `handoff-read-196`

Approved-by:

## Why

In the 2026-10-26 Codex dogfood pass, MCP pickup as `agent:codex` printed `Last Handoff: none for codex@main` and only the key and age of `claude@main`. The Claude slot held the milestone, next issues and constraints; getting them took a large state dump (#196). `writePickupHandoffs` (`internal/state/state.go:2693`) prints other same-checkout slots through `formatOtherHandoffLine` (`state.go:2800`) with no goal, and nothing in the CLI or MCP reads a chosen slot's five fields without also running pickup, which writes the cache (`ReconcilePickup`, `state.go:1181-1214`).

## What Changes

- Each `other handoffs:` pickup line gains a bounded goal preview after its attribution: `other handoffs: <agent>@<branch> (<age>[, PR #<n>]): <goal>`, the goal clipped by the existing `cleanTaskText` (104 bytes).
- A read-only path prints one named slot's five fields: CLI `ateam state read-handoff <agent>@<branch>` and MCP `ateam_state` action `read_handoff` with a `slot` string. It reads the current checkout's cache, then linked worktrees' caches, and never writes the cache, touches the writer lease, changes caller identity, or posts to a forge.
- A stale slot is printed under a `STALE — <reasons>` banner; a missing slot exits 1 (MCP tool error) and lists the slots that exist.
- `docs/mcp.md` and `ateam state --help` document the new action and subcommand; the harness spec gains one Requirement.

## Track

`design`. It adds a spec Requirement and exported surface reachable from an adopting repository (a CLI subcommand and an MCP action). See `design.md`.

## Non-goals

- Taking over, claiming, or copying another slot's work; no change to slot isolation (#72) or the write refusal.
- Changing which slots pickup prunes, or the worktree `handoff:` lines (#136).
- A bare-agent shorthand (`codex` meaning `codex@<current branch>`).

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor` parity green; `TestMCPTokenBudget` under 300 (212 today).
- Regression: seed a `claude@main` handoff, query as Codex; the pickup preview and `read_handoff` both show the Claude goal, and the cache bytes, writer lease and Codex identity are unchanged afterwards. Missing and stale slots print their stated outputs.
- `task mutate:check` reports `detection` for the preview, the read-only guarantee, and the stale banner.
- `TestHookLatencyBudget` passes: the preview adds string formatting only, no I/O.
