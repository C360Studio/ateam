## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#313).
  Hold: OPEN-Q proposal awaits owner approval

## 2. Fix

- [x] 2.1 `reviewRequest` (`cmd/ateam/reviewlaunch.go`) reads `head_sha` from `readPullBody` whether or not `--implementer` is given, and exits 2 naming the PR's head when it cannot read the PR under `--implementer` (#313). Acceptance: `TestReviewRequest` subtests `implementer-flag-takes-pr-head` and `offline-with-implementer-flag`; `task mutate:check` reports `detection` for the pre-fix file and for the refusal disabled. Evidence: `go run ./cmd/ateam mutate` reports `detection` for both mutants.
- [x] 2.2 `docs/adoption.md` states that the head comes from the PR (#313). Acceptance: `task spec:check` passes.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#313).
