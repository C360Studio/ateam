# Change Proposal: `task-lease-284`

Approved-by:

## Why

An adopting factory daemon runs ateam in its worker sandboxes and on developer workstations. It holds a task lease from its own server, which is a different lease with a different owner from ateam's writer lease. Agents inside the checkout cannot see that task lease, a handoff cannot be checked against it, and the daemon has to grep a warning line to learn whether a checkout has a live writer (#284, owner ruling 2026-11-03).

## What Changes

- Pickup and session-start read `.ateam/task-lease.json` (override `ATEAM_TASK_LEASE_FILE`) and print one `Task lease:` line, with `expired` appended when `expires_at` is past. A missing file prints nothing. An unreadable one prints one `WARNING: task lease file unreadable (<reason>)` line and pickup continues.
- A current handoff whose `Addresses` carries `aip-lease:<id>` prints one line when `<id>` differs from the file's `id`.
- `ateam state writer --json` prints an object with `state` (`none`, `live`, `stale`, `unknown`), `reason`, and the stored writer fields. All four states exit 0; bad usage exits 2.
- The writer lease records `host` and, on Linux, `pid_ns`, so a PID check is trusted only on the host and namespace that wrote it.
- `ateam version --json` prints the version, commit, and a `capabilities` list, so a client can tell an old ATeam from a failed command.
- The docs say what the check is worth: advisory, with no enforced yield. A daemon gives each task its own worktree.
- `openspec/specs/harness/spec.md` gains separate optional Requirements, as the herdr adapter has. The herdr Requirement's "byte-for-byte unchanged" wording narrows to herdr identity, since the lease gains two fields everywhere.

## Track

`design`: the file is a contract with a daemon outside this repository and `writer --json` is surface reachable from an adopting repository. See `design.md`.

## Non-goals

- ateam never writes, refreshes, or clears the file, and never calls the daemon or its server. No heartbeat, no second lease.
- The task lease changes no ownership decision: not the writer lease, the handoff slot, or prune.
- No MCP tool, and no board or UI change (#273 reads the file later).
- No enforcement: `before-prompt` and `turn-end` still overwrite a foreign lease, and pickup's own PID-based warning is unchanged (separate issue).

## Definition of Done

- Tests with attached spec references for: file fresh, expired, absent, malformed; Addresses token match and mismatch; every `writer --json` row in the D7 table; `version --json`.
- `task mutate:check` reports `detection` on the expiry comparison, the token parser, and each D7 branch.
- `ateam doctor` hook latency stays inside the 10ms target with the two new lease fields.
- `docs/hook-flow.md` names the file and the override; `docs/adoption.md` gives the daemon's seam.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
