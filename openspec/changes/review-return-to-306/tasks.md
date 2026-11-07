## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#306). Evidence: the owner asked in Enjoy on 2026-11-07 to fix #306 now on the quick track, approving the fix the issue body states; Claude wrote the header at their request.

## 2. Guard

- [x] 2.1 `reviewRequest` (`cmd/ateam/reviewlaunch.go:203-214`) refuses a launch whose manifest names `{return_to}` when `spec.ReturnTo` is empty, warning only under `--dry-run` (#306). Acceptance: `TestReviewRequest` subtests `return-to-*`; `task mutate:check` reports `detection` with the check disabled and with the dry-run exception removed. Evidence: subtests `return-to-command-without-return-to`, `return-to-tool-without-return-to`, `return-to-dry-run-warns`, `return-to-host-env-id`, `return-to-manifest-without-placeholder`; `go run ./cmd/ateam mutate` reports `detection` for both mutants.
- [x] 2.2 `docs/adoption.md` states the refusal under the host table (#306). Acceptance: `task spec:check` passes. Evidence: the paragraph after the host table.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#306).
