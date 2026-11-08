## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#321). Hold: OPEN-Q proposal awaits owner approval

## 2. Fix

- [ ] 2.1 `PruneVerdict` (`internal/state/prune.go`) runs `occupiedBy` for a detached review checkout of a closed PR unless `force` (#321). Acceptance: a `PruneVerdict` test covers another session's lease and handoff (kept), the caller's own lease and `--force` (removed); `task mutate:check` reports `detection` with the check removed.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#321).
