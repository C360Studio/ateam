## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#321). Evidence: the owner selected "Approve, go ahead" in Enjoy on 2026-11-08; Claude wrote the header at their request.

## 2. Fix

- [x] 2.1 `PruneVerdict` (`internal/state/prune.go`) runs `occupiedBy` for a detached review checkout of a closed PR unless `force` (#321). Acceptance: a `PruneVerdict` test covers another session's lease and handoff (kept), the caller's own lease and `--force` (removed); `task mutate:check` reports `detection` with the check removed. Evidence: `ateam mutate -pkg ./internal/state -test TestPruneKeepsReviewCheckoutAnotherSessionOccupies -file internal/state/prune.go -mutant <pre-fix prune.go> -expect-text 'holds the lease (1m ago)'` reported `detection`, SHA-256 of `prune.go` unchanged.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#321).
