# Tasks: gate-approval-delegation-453

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-28) sha256:ff7c9a41ec68`).

## 2. Implementation

- [x] 2.1 `.agents/protocol.md:94` and `:101`: the developer run that claimed the PR posts the comment, then the label, on the owner's own yes naming the PR and its live head; a relayed or stale yes is not one; ask last (after the approving review, the archive commit and an update from `main`). Acceptance: the owner-only sentences are gone and the shared-login attestation sentence stays.
- [x] 2.2 `.agents/contracts/developer.md:7` and `developerContract` (`internal/bootstrap/review.go:23`): the steps from D2–D4. Acceptance: byte parity in `TestCanonicalReviewContracts` (`review_test.go:151`) and a new literal for the gate-file paragraph.
- [x] 2.3 `.agents/contracts/coordinator.md:5`, `:7` and `coordinatorContract` (`templates.go:368`, `:370`): still never posts; the owner message links the developer's session. Keep the literal at `coordinator_test.go:43`.
- [x] 2.4 `.agents/skills/coordinate/SKILL.md:26` and `templates.go:421`: for a gate-file change, the PR's live head and a link to the developer's session. Acceptance: byte parity in `TestInitShipsCoordinatorAsCanonical`.
- [x] 2.5 `AGENTS.md:91`, `.agents/rules-index.md:42`, `docs/adoption.md:176`, `docs/adr/0003-roles-and-routes.md:26`: the delegated post in each.
- [x] 2.6 New literals in `TestInitShipsGateApprovalDelegation` (`coordinator_test.go`; `review_test.go` keeps its byte parity), shown able to fail: three `task mutate:check` mutants (`review.go`, two in `templates.go`), each `detection`.
- [x] 2.6a Asking last (D4) leaves `Gate files` red at ready-for-review, which the #446 rule (`developer.md:9`, `SKILL.md:22`) would treat as a failing check; both now exempt it on a gate-file PR, pinned by the same test.
- [ ] 2.7 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
