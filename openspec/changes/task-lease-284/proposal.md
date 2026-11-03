# Change Proposal: `task-lease-284`

Approved-by: Coby Leuschke (2026-11-03)

## Why

An adopting factory daemon runs ateam in its worker sandboxes and on developer workstations. It holds a task lease from its own server, which is a different lease with a different owner from ateam's writer lease. Agents inside the checkout cannot see that task lease, a handoff cannot be checked against it, and the daemon has to grep a warning line to learn whether a checkout has a live writer (#284, owner ruling 2026-11-03).

## What Changes

- Pickup and session-start read `.ateam/task-lease.json` (override `ATEAM_TASK_LEASE_FILE`) and print one `Task lease:` line, with `expired` appended when `expires_at` is past. A missing file prints nothing. An unreadable one prints one `WARNING: task lease file unreadable (<reason>)` line and pickup continues.
- A current handoff whose `Addresses` carries `aip-lease:<id>` prints one line when `<id>` differs from the file's `id`.
- `ateam state writer --json` prints the writer lease as stored plus `live` and `exited`, or `null`, with exit 0 for no lease.
- `openspec/specs/harness/spec.md` gains separate optional Requirements, as the herdr adapter has, so every existing scenario stays unchanged when the file is absent.

## Track

`design`: the file is a contract with a daemon outside this repository and `writer --json` is surface reachable from an adopting repository. See `design.md`.

## Non-goals

- ateam never writes, refreshes, or clears the file, and never calls the daemon or its server. No heartbeat, no second lease.
- The task lease changes no ownership decision: not the writer lease, the handoff slot, or prune.
- No MCP tool, and no board or UI change (#273 reads the file later).

## Definition of Done

- Tests with attached spec references for: file fresh, expired, absent, malformed; Addresses token match and mismatch; `writer --json` for a foreign live lease, an exited PID, and no lease.
- `task mutate:check` reports `detection` on the expiry comparison, the token parser, and the liveness rule.
- `docs/hook-flow.md` names the file and the override; `docs/adoption.md` gives the daemon's seam.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
