# Design: writer-decision-287

## Context

- `internal/hook/hook.go:152-156`: session-start builds the summary, then writes the lease unless the text contains `\nWARNING: another writer may be active`. #284 required the leading newline so a task lease value cannot match; a cached decision or handoff field that holds a newline still can (#287).
- `internal/state/state.go:3120-3145`: the warning needs `IsProcessAlive(st.Writer.PID)`. A PID from another host or container means nothing here, so such a lease prints `(exited)` and no warning (#286).
- `internal/state/writerstatus.go:46-78`: `classifyWriterLease` checks place first (`Host`, `PIDNamespace` against `processPlace()`), then the PID. Spec `Machine-readable writer state` (harness spec, line 961) already counts a lease with no host as from elsewhere.
- `internal/hook/hook.go:104-108`: `before-prompt` calls `refreshWriterLease`, which writes the lease unconditionally for any non-read-only session. A session that session-start kept off the lease takes it on its first prompt.
- `internal/state/state.go:709`: `writerLeaseTTL` is 10 minutes. `WriteWriterLease` (`state.go:1479`) records `Host` and `PIDNamespace` since #284.

## Goals / Non-Goals

- Goal: one decision, read by both the summary and the hook; a recent lease from elsewhere counts as another writer.
- Non-goals: a liveness probe across hosts; changing `ateam state writer --json` or the handoff release rule.

## Decisions

### D1. Where the decision lives

- **A (recommended):** an exported function in `internal/state`, say `CompetingWriter(st *State) *WriterLease`, that `BuildPickupSummary` and the hook both call. Cost: one function and a small refactor of the summary block. Forecloses nothing.
- **B:** `BuildPickupSummary` returns a struct with the text and the decision. Cost: changes a signature used by the MCP server, `ateam state pickup` and tests. No gain over A.
- **C:** keep the text search with a sentinel that no cached text can carry. Still parses output; rejected.
- Adopter seam: none; output text is unchanged for a local lease.

### D2. A lease from elsewhere

- **A (recommended, the owner ruling):** reuse the place check from `classifyWriterLease`. From elsewhere and within the TTL, warn without a PID check; past the TTL, no warning. No host counts as elsewhere, as in the writer spec. The writer line shows the place reason instead of `(exited)`.
- **B:** treat a lease with no host as local and PID-check it. Keeps today's behaviour for pre-#284 leases, but disagrees with `ateam state writer --json`. Such leases age out in 10 minutes either way.
- Adopter seam: two sessions in different containers on one checkout now see the warning and the second stays off the lease, as ruled. A developer who does nothing gets this on upgrade.

### D3. Does `before-prompt` respect the decision? (owner ruling needed)

- **A:** leave `before-prompt` unchanged and file a follow-up issue. Smallest change, but the warned session takes the lease on its first prompt, so #286's ruling holds only until then.
- **B (recommended):** `before-prompt` refreshes the lease only when D1 finds no competing writer. `handle` has already loaded the state, so this adds no file read. A warned session takes the lease once the other one hands off or goes 10 minutes without a prompt. Cost: a second session in one checkout never holds the lease while the first is active; that is the ruled intent.
- What B forecloses: a session cannot take over a live lease just by being used; it needs the other session's handoff, or the TTL to run out.

## Risks / Trade-offs

- A crashed writer elsewhere holds off a new session for up to 10 minutes. Accepted by the ruling; the warning says so, and a handoff clears it.
- Under D3 B, a session whose own `before-prompt` is the only refresher keeps its lease, since D1 excludes the same session.

## Migration Plan

None. The lease file format is unchanged.

## Open Questions

- D3 needs an owner ruling (A or B).
