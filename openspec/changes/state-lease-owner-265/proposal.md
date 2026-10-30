# Change Proposal: `state-lease-owner-265`

Approved-by:

## Why

When session B saves a handoff, it deletes session A's live writer lease, through both the CLI and MCP, and even when both sessions belong to the same agent (#265). `cmd/ateam/main.go:351,361` and `internal/mcpserver/server.go:327,332` set `cur.Writer = nil` and call `state.ClearWriterLease` without any condition. `ClearWriterLease` (`internal/state/state.go:1365`) removes `.ateam/writer.json` without checking who owns it. The fresh-handoff guard protects the handoff slot, not the lease. Once A's lease is gone, pickup stops warning that another writer is active, and prune (#263) can no longer rely on the lease.

## What Changes

1. **Handoff releases only the caller's own lease.** One new `state` function decides ownership, saves the handoff, and removes the lease, all under the existing `.ateam/state.lock` (`WithLock`, `state.go:1588`). The CLI and MCP handoff paths both call it. A foreign lease, or one held by an unknown identity, stays in place, and the cached `Writer` keeps that lease (`design.md` D1, D2).
2. **Lease writes take the same lock.** `WriteWriterLease` (`state.go:1374`) takes `WithLock`, so a replacement lease written during a handoff cannot be deleted after the ownership check (`design.md` D3).
3. **Release errors are reported.** If the handoff saves but the lease removal fails, the CLI exits 1 and MCP returns an error, both saying the handoff was saved and the lease was not released. No `_ =` remains (`design.md` D4).
4. Spec: a new harness Requirement, "Handoff releases only the caller's writer lease".

## Track

`design`. It changes durability behavior (who may delete the writer lease) and adds a spec Requirement. See `design.md`.

## Non-goals

- Prune's occupancy policy (#263), how session or agent identity is resolved, and a `--force` override for the lease.
- The `before-prompt` hook's unconditional lease rewrite (`internal/hook/hook.go:158`). It overwrites rather than deletes, and it is how a second session takes over the lease. It was seen in the 2026-10-30 dogfood pass and gets its own issue if the owner wants one.
- Stale or dead foreign leases: handoff leaves them alone. Pickup already treats them as non-competing.

## Definition of Done

- The #265 reproduction, scripted as tests in a disposable repo: B's CLI handoff and B's MCP handoff each leave A's live `writer.json` byte-identical and the cached writer equal to it. This holds for a different agent and for the same agent with a different session.
- A caller's own lease is still removed on a successful handoff. A refused handoff (slot held) releases nothing.
- A synchronized test: a replacement lease written while the handoff holds the lock (a test hook inside the critical section, no sleeps) survives.
- `task mutate:check` reports `detection` when the ownership check is replaced with `true`. Examples are enough here: the decision has two inputs (lease, caller), and the history is a single write followed by a single release.
- A release error surfaces as exit 1 or an MCP error naming the saved handoff.
- `task verify` is green on a clean tree, `ateam doctor` parity is green, `ateam bench --selftest` is green, and hook latency stays within budget.
