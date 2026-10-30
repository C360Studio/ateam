# Change Proposal: `prune-occupancy-263`

Approved-by: Coby Leuschke (2026-10-30)

## Why

`ateam state prune` decides ownership by branch prefix (`PruneVerdict`, `internal/state/prune.go:21`), and every session of one agent shares that prefix. On 2026-10-29 a dry run listed `claude/merge-check-workflow-255` as `would remove: merged via #260` while that worktree's own `.ateam/state.json` held a handoff another Claude session had written one minute earlier (#263). The only occupancy guard is the caller's own checkout, so one session can remove the checkout another is standing in.

## What Changes

1. Before `PruneVerdict` clears a finished branch worktree, it reads that worktree's `.ateam/writer.json` and `.ateam/state.json`. A lease whose pid is alive, or whose `last_seen` is within the writer-lease TTL (10m), keeps it: `session <id> holds the lease (<age> ago)`. A handoff for the worktree's branch in its own cache fresher than 30 minutes keeps it: `session <id> handed off <age> ago; prune after 30m or pass --force`.
2. The caller's own session never blocks itself. The caller is `st.CurrentSessionID`, else the session env vars, else the caller's live lease in its own checkout (the existing `ResolveActiveSessionID` order). When no caller session resolves, every lease and handoff counts.
3. `ateam state prune --force` skips only these two rules; `--force --dry-run` prints what `--force` would remove. Dirty, ownership, and unmerged rules are unchanged.
4. Pickup's `, prune candidate` label calls the same verdict, so the label and the command agree.
5. Spec: a scenario added to the harness Requirement "Prune finished worktrees the caller owns"; README and `ateam state` help name `--force`.

## Track

`quick`. One package (`internal/state`) plus the CLI flag. It adds a scenario but no Requirement, no exported surface reachable from an adopting repository, and touches no boot path, durability behavior, or gate file. The owner chose this on 2026-10-30.

## Non-goals

- Occupancy checks for detached `<agent>-review-pr-<n>` checkouts; their removal stays tied to the PR's state.
- Whether the owner may prune other agents' finished worktrees (`--agent all`); a protocol policy call.
- A `--session` flag on `ateam state prune`, or a `force` field in the MCP schema (prune is not an MCP action).

## Definition of Done

- `TestPruneKeepsWorktreeAnotherSessionOccupies` covers another session's live-pid lease, recent and expired leases without a live pid, a 5-minute and a 2-hour handoff, the caller's own lease and handoff, an unknown caller, and `--force` past a lease and a handoff, through `PruneVerdict`, the `Prune` dry run, and the pickup label.
- `task mutate:check` reports `detection` when the lease read is dropped.
- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
