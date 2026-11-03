## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#284). Evidence: the owner approved in Enjoy on 2026-11-03 selecting "Approve as drafted (D7 A, liveness-only `live`)"; Claude wrote the header at their request.

## 2. Task lease

- [x] 2.1 `ReadTaskLease(root)` and the D2 validation; `Load` (`internal/state/state.go:1586`, `:1614`) fills `st.TaskLease` and `st.TaskLeaseError` (`json:"-"`); `BuildPickupSummary` (`:2754`) prints the line or warning after the writer block. Acceptance: tests for fresh, expired, absent, each malformed shape, a FIFO, and the override path; `task mutate:check` reports `detection` with the expiry comparison inverted (#284). Evidence: `internal/state/tasklease.go`, `State.TaskLease`, `writePickupTaskLease`; `TestTaskLeaseLine`, `TestTaskLeaseMalformedWarnsOnce` (13 malformed shapes, a directory, limits exact), `TestTaskLeaseFIFODoesNotBlock`, `TestStatePickupPrintsTaskLeaseThroughRefresh`. `go run ./cmd/ateam mutate` reports `detection` with the expiry comparison inverted, the regular-file check disabled, and `unicode.IsControl` disabled.
- [x] 2.2 `hook.go:143` matches the warning at line start (D4). Acceptance: a session-start test with the warning text as `holder` still writes the writer lease; `task mutate:check` reports `detection` with the old match restored (#284). Evidence: `TestSessionStartTaskLeaseTextCannotGateWriterLease`; `detection` with the old match restored.
- [x] 2.3 Addresses token check (D5). Acceptance: tests for mismatch, match, case-insensitive prefix, no token, no file; `task mutate:check` reports `detection` with the prefix made case-sensitive and with the comparison inverted (#284). Evidence: `TestTaskLeaseAddressesToken`; `detection` with `(?i)` removed and with the comparison inverted.

## 3. Writer JSON

- [x] 3.1 Extract `writerLeaseLive` from `isCallerActiveWriterLease` (`state.go:671-680`) with existing callers unchanged (D7). Add `runStateWriter` (`cmd/ateam/main.go:129`, `stateUsage` `:168`). Acceptance: CLI tests for a foreign live lease, an exited PID, no lease, a corrupt lease, and no `--json`; `task mutate:check` reports `detection` with the TTL check and the PID check each removed (#284). Evidence: `writerLeaseLive`, `ReadWriterLeaseStatus`, `runStateWriter`; `TestWriterLeaseLive`, `TestStateWriterJSON`. `detection` with the TTL check removed, the PID check removed, and `live` computed by `isCallerActiveWriterLease` as the ruling first read.

## 4. Docs and verify

- [x] 4.1 `docs/hook-flow.md` names the file and the override; `docs/adoption.md` gives the D1 seam, naming no adopter (#284). Acceptance: `ateam doctor --check` and `task spec:check` pass. Evidence: `docs/hook-flow.md` paragraph; `docs/adoption.md` § Optional external task lease.
- [ ] 4.2 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#284).
