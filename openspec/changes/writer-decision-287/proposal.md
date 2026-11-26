# Proposal: writer-decision-287
Approved-by:

## Track

`design`: changes who takes the writer lease at session start (a boot path and ownership behaviour), and adds a harness Requirement. Ships #287 and #286 as one PR, per the owner decision of 2026-11-26.

## Decisions you are making by approving

- Session start decides whether to take the writer lease from the lease itself, not from the text it prints, so no issue decision, handoff field or other cached text can stop a session from taking the lease (#287).
- A recent lease written on another host or in another container raises the "another writer may be active" warning, and the new session does not take the lease, even though ateam cannot check that process is alive (#286, owner ruling of 2026-11-26).
- A lease that names no host (written before #284) counts as from elsewhere, matching `ateam state writer --json`. It stops counting once it is 10 minutes old.
- D3 below: whether the first prompt of a warned session may still take the lease. Today it does, which undoes the session-start decision on the first prompt.

## Why

`session-start` takes the lease unless the rendered summary contains `\nWARNING: another writer may be active` (`internal/hook/hook.go:154`), so a cached decision holding that text on its own line suppresses the lease (#287). The summary gates the warning on a local PID check (`internal/state/state.go:3133-3142`), so a writer in another container reads as exited and gets overwritten (#286). `classifyWriterLease` (`internal/state/writerstatus.go:46-78`) already tells a lease from elsewhere apart; the summary does not use it.

## What Changes

1. One function in `internal/state` decides whether another writer is active. `BuildPickupSummary` prints the warning from it and the hook reads it; the hook no longer searches text.
2. That function skips the PID check for a lease from elsewhere and warns while it is within the TTL. The writer line names the place (`other host`, `other pid namespace`, `no host recorded`) instead of `(exited)`.
3. Per D3, `before-prompt` either keeps refreshing the lease unconditionally or applies the same decision.

## Scope & Non-Goals

- **In-bounds:** `internal/state/state.go`, `internal/state/writerstatus.go`, `internal/hook/hook.go` and their tests; one ADDED Requirement in `openspec/specs/harness/spec.md`.
- **Non-goals:** a liveness probe across hosts; changing `ateam state writer --json`; the handoff's release rule; the task lease; read-only review sessions (unchanged).

## Definition of Done

- Hook tests: a cached decision and a handoff field holding the warning text on its own line still let session-start write the lease; a lease from another host within the TTL warns and is not overwritten, past the TTL it is; a local exited PID is overwritten as today.
- `task mutate:check` detects reverting the hook to the text search and restoring the PID check for a lease from elsewhere.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` green; the change archived before ready.
