# Design: `state-lease-owner-265`

## Context

Read against `origin/main` `d7bb142` (v0.3.4).

- `cmd/ateam/main.go:346-361`: `runStateHandoff` runs `state.Update` → `SetHandoffGuarded`, sets `cur.Writer = nil`, then calls `_ = state.ClearWriterLease(repoRoot)` after the lock is released. The caller's session comes from `state.ResolveActiveSessionID` (`state.go:1314`): `--session`, then the env, then the lease's own session, but only when `isCallerActiveWriterLease(wl, "")` proves the PID.
- `internal/mcpserver/server.go:322-332`: the same pattern. The session comes from `resolveEffectiveSessionID` (`:169`): an explicit arg, then the env, then the lease session when the lease PID is this client's or the server's.
- `internal/state/state.go:1365` `ClearWriterLease`: an unconditional `os.Remove`. `:1374` `WriteWriterLease` → `writeWriterLeaseStruct` (`:1481`): temp file plus rename, **without** `WithLock`.
- `internal/state/state.go:672` `isCallerActiveWriterLease(lease, callerSession)`: false when the lease is empty, the PID is dead, or `last_seen` is 10 minutes old or more. With a caller session it needs an exact session match. With none it matches by herdr pane or client/own PID. With none of those, it returns `leaseSession == ""`.
- `internal/state/state.go:1541-1543` `Load`: `st.Writer` always comes from `writer.json`, so the cached writer cannot drift from the file between loads.
- `internal/state/state.go:1588` `WithLock`: an exclusive `flock` on `.ateam/state.lock`. The hook already takes it through `state.Update` (`internal/hook/hook.go:169,198,278,323`). No `Update` callback does network I/O.
- Writers of the lease: the hook only (`hook.go:144,158,276`). Removers: the two handoff paths only.

## Goals / Non-Goals

Goals: a handoff deletes only the caller's own lease, cannot race a concurrent lease write, and reports a failed release. Non-goals: see `proposal.md`.

## Decisions

### D1. The ownership predicate

- **A. Reuse `isCallerActiveWriterLease` as-is.** No new code. The last branch, though, lets a caller with no session clear a lease that has no session, PID, or pane: unknown identity authorizes clearing.
- **B (recommended). Reuse it, and also require a positive identity match.** Release only when the predicate holds and the match came from a session (both non-empty and equal), a herdr pane, or the client/own PID. A lease with no identity is never cleared by handoff. Cost: one helper, `ownsWriterLease`, in `state.go` (about 10 lines). `ResolveActiveAgent` keeps its current behavior.
- **C. Session equality only, with no liveness check.** This releases the caller's own stale lease too, but drops the PID and herdr paths, so a session-less CLI caller can never release its lease.

A dead or stale lease fails the predicate under A and B and is left in place. Pickup already treats such a lease as non-competing, and the next hook overwrites it.

### D2. One critical section in `state`

Add `state.SaveHandoffReleasingLease(root, h, force, callerSession, mutate func(*State)) (released bool, err error)`, or a narrower name if the reviewer prefers. Inside `WithLock` it:

1. Loads.
2. Runs `SetHandoffGuarded`. On refusal it returns, and nothing is saved or released.
3. Applies `mutate` (the `CurrentAgent` assignment).
4. Reads `writer.json` and decides ownership (D1).
5. If owned, sets `Writer = nil`.
6. Saves.
7. If owned, removes `writer.json`.

The CLI and MCP each drop their `Update` block and the trailing `ClearWriterLease`, and call this function instead. `ClearWriterLease` loses its last caller and is deleted (exported, but internal-only: `internal/state` is not importable by adopters).

The order is save, then remove. If the remove fails, the handoff is still recorded, and `Load` re-reads the surviving lease, so cache and file agree on the next read.

### D3. Lease writes take the lock

`WriteWriterLease` wraps `writeWriterLeaseStruct` in `WithLock`. Then no lease can be written between step 4 and step 7 of D2.

- Hook cost: one uncontended `flock`, a few microseconds. When contended, it waits for the longest `Update` callback, all of which are local file I/O. `hook.go:158` (`before-prompt`) gains the lock. `session-start` already holds it nearby.
- Rejected: compare-and-delete without the lock (rename to a temp name, compare, rename back). Renaming back could clobber a lease written in the meantime, and it adds a second protocol.

### D4. Reporting a failed release

The CLI prints `ateam state handoff: handoff saved; writer lease not released: <err>` and exits 1. MCP returns that text with `isError: true` and skips the forge post. A foreign lease left in place is not an error. The CLI is silent about it, and the MCP status gains `writer lease: kept (held by <agent> session <id>)` so the caller sees why pickup still warns.

## Test plan

- A table test over `ownsWriterLease`: same session; different session with the same agent; different agent; no caller session with a matching client PID; matching herdr pane; dead PID; stale `last_seen`; a lease with no identity. Each row has an expected bool written from the D1 rules, not computed by the code under test.
- The CLI (`runStateHandoff`) and MCP (`callTool`) reproductions from #265 in `t.TempDir()` repos, using the test process's PID as session A's live process. Assert `writer.json` is byte-identical and the cached `Writer` equals it.
- Own-lease release, and a refused handoff (a fresh foreign slot) that leaves the lease in place.
- Race: two nil-in-production test seams. `afterLeaseDecision` runs between steps 4 and 7, and `beforeFlock` runs in `WithLock` just before the `flock` call. The test's `afterLeaseDecision` starts a goroutine that calls `WriteWriterLease`, then waits on `select` for either the goroutine's `beforeFlock` signal (it is about to block on the lock) or its completion (it wrote without the lock). After the handoff returns, the test waits for the goroutine and asserts that the replacement lease exists. The test decides the same way on every run: with D3 the goroutine always blocks and its write survives, and without D3 it always writes before the remove, so the write is deleted and the test fails. No sleeps.
- Failure: make `.ateam/writer.json` impossible to remove (a non-empty directory at that path) → exit 1 and an MCP error. The handoff slot is saved.
- Mutation: replace the `ownsWriterLease` call with `true` → `detection`.

## Risks / Trade-offs

- The hook can now block behind a long `Update`. All callbacks are local, and `doctor` keeps measuring `hook_latency_us`.
- A session that never resolves a session ID or PID can no longer release its lease by handing off. Its lease ages out after 10 minutes (D1 B). This is intended: unknown identity is not authority.

## Open Questions

1. D1: A, B (recommended), or C?
2. D4: should a kept foreign lease print a stderr note on the CLI too, or stay silent as recommended?
