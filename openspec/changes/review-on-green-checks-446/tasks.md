# Tasks: review-on-green-checks-446

## 1. Approval

- [x] 1.1 Owner approves the proposal.

## 2. Implementation

- [x] 2.1 Developer contract defines a `ready` handoff (premise: `.agents/contracts/developer.md` has no rule on check state at handoff; its copy is `developerContract`, `internal/bootstrap/review.go:17`, kept equal by `review_test.go:152`). Accept: the paragraph in both copies.
- [x] 2.2 Coordinate step 3 reads the required checks before requesting review (premise: step 3 runs `ateam review request` on any ready-for-review handoff, `.agents/skills/coordinate/SKILL.md` § 3 and `internal/bootstrap/templates.go:417`; the ruleset requires `Required`, `Gate files` and `Merge check`, and `Merge check` is absent until the `reviewed-by:` line, protocol § 4). Accept: the sentence in both copies.
- [x] 2.3 Shipped-text test in `internal/bootstrap/coordinator_test.go` citing the new scenario; mutation of each template string shown as `detection`. Harness spec delta applied; `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
