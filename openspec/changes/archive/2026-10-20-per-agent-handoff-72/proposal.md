# Change Proposal: Per-Agent, Per-Branch Handoff Slots (`#72`)

## Why
A single `last_handoff` slot in `.ateam/state.json` gets overwritten whenever any agent or `pre-compact` hook saves a checkpoint, losing concurrent handoffs across agents and branches and surfacing another agent's handoff as the current session's `Last Handoff`.

## What Changes
- Store handoffs in `.ateam/state.json` under `handoffs: map[string]*Handoff` keyed by `<agent>@<branch>`, while maintaining `last_handoff` as an alias for the current session's `<agent>@<branch>` key.
- Evaluate staleness rules (`#39`/`#53`) per slot in `RefreshHandoffStaleness()` and drop stale entries from `handoffs` on pickup.
- Render the current `<agent>@<branch>` handoff first (`Last Handoff:\n...` or `Last Handoff: none for <agent>@<branch>`), followed by sorted `other handoffs: <agent>@<branch> (<age>[, PR #<n>])` lines.
- Accept `--agent` / `agent` in `ateam state handoff`, `ateam state pickup`, and `ateam_state`, defaulting to `--agent`, `ATEAM_AGENT`, active writer lease, or branch prefix.

## Impact
- `internal/state/state.go`, `internal/state/state_test.go`
- `internal/hook/hook.go`, `internal/mcpserver/server.go`, `internal/web/server.go`, `cmd/ateam/main.go`
- `openspec/specs/harness/spec.md`
