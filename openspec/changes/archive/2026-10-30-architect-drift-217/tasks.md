## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#217). Evidence: on 2026-10-30 the owner picked "Yes, same paragraph check as developer/reviewer" in Enjoy; Claude wrote the header from that answer.

## 2. Implementation

- [x] 2.1 Add `.agents/contracts/architect.md` to `reviewFiles` (`internal/bootstrap/review.go:140`). Acceptance: drift, conflict, and lifecycle tests cover `architect.md`; `task mutate:check` dropping the entry reports `detection` (#217). Evidence: `TestReviewSetupDrift/.agents/contracts/architect.md`, `TestConfigureReviewConflictNoWrites/.agents/contracts/architect.md`, and the lifecycle test's appended architect paragraph; `ateam mutate` with the map entry removed reported `detection` (`review_test.go:120`), SHA-256 of `review.go` unchanged.
- [x] 2.2 Update `docs/brownfield.md` § Existing specialist contracts (#217). Evidence: the base-contracts bullet names doctor and all three contracts; the base-name bullet drops "until #217".
- [x] 2.3 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green. Evidence: `task verify` exit 0 on a clean tree at `89b9e56` (`test-spec: all discovered supported tests have valid citations`, `selftest_passed: true`); `./bin/ateam doctor --check` `passed: true`, `violations: null`; `./bin/ateam bench --selftest` `selftest_passed: true`.
- [x] 2.4 Archive the change before marking the PR ready (#217). Evidence: archived to `openspec/changes/archive/2026-10-30-architect-drift-217/`.
