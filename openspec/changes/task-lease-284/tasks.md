## 1. Approval

- [ ] 1.1 Owner approves the revised `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#284). Earlier approval of the first draft: 2026-11-03; the owner directed this revision the same day after the adopter's feedback. Hold: OPEN-Q proposal awaits owner approval

## 2. Task lease

- [x] 2.1 `ReadTaskLease(root)` and the D2 validation; `Load` (`internal/state/state.go:1586`, `:1614`) fills `st.TaskLease` and `st.TaskLeaseError` (`json:"-"`); `BuildPickupSummary` (`:2754`) prints the line or warning after the writer block. Acceptance: tests for fresh, expired, absent, each malformed shape, a FIFO, and the override path; `task mutate:check` reports `detection` with the expiry comparison inverted (#284). Evidence: `internal/state/tasklease.go`, `State.TaskLease`, `writePickupTaskLease`; `TestTaskLeaseLine`, `TestTaskLeaseMalformedWarnsOnce` (13 malformed shapes, a directory, limits exact), `TestTaskLeaseFIFODoesNotBlock`, `TestStatePickupPrintsTaskLeaseThroughRefresh`. `go run ./cmd/ateam mutate` reports `detection` with the expiry comparison inverted, the regular-file check disabled, and `unicode.IsControl` disabled.
- [x] 2.2 `hook.go:143` matches the warning at line start (D4). Acceptance: a session-start test with the warning text as `holder` still writes the writer lease; `task mutate:check` reports `detection` with the old match restored (#284). Evidence: `TestSessionStartTaskLeaseTextCannotGateWriterLease`; `detection` with the old match restored.
- [x] 2.3 Addresses token check (D5). Acceptance: tests for mismatch, match, case-insensitive prefix, no token, no file; `task mutate:check` reports `detection` with the prefix made case-sensitive and with the comparison inverted (#284). Evidence: `TestTaskLeaseAddressesToken`; `detection` with `(?i)` removed and with the comparison inverted.

## 3. Writer state

- [ ] 3.1 Remove the draft's `writerLeaseLive`, restoring `isCallerActiveWriterLease` (`state.go:671`). `WriteWriterLease` (`:1465`) records `host` and `pid_ns` (D8). Acceptance: `TestWriterLeaseHerdrIdentity` and every `assertLeaseKept` caller pass; `ateam doctor` hook latency reported before and after (#284).
- [ ] 3.2 `runStateWriter` (`cmd/ateam/main.go`) prints the D6 object from the D7 table. Acceptance: a CLI test per table row with the expected `state` and `reason` taken from the table, plus bad usage; `task mutate:check` reports `detection` with each of the host, `pid_ns`, TTL, and PID checks removed (#284).
- [ ] 3.3 `ateam version --json` (D9, `main.go:69`). Acceptance: tests for the JSON form and for plain `version` text unchanged (#284).

## 4. Docs and verify

- [ ] 4.1 `docs/hook-flow.md` names the file and the override; `docs/adoption.md` gives the D1 seam, the D7 table, the D9 check, and the D10 advisory rule, naming no adopter (#284). Acceptance: `ateam doctor --check` and `task spec:check` pass.
- [ ] 4.2 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#284).
