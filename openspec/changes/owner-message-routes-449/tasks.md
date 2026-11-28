# Tasks: owner-message-routes-449

## 1. Approval

- [x] 1.1 Owner approves the proposal.

## 2. Implementation

- [ ] 2.1 Coordinate step 4 lists each gate with the developer's conversation link first and the PR second, offers the three approval routes (dropping one a developer reported refused), reads the branch and PR before asking, and counts a repeated gate once (premise: step 4 lists "the PR or issue, the question in plain words, and what you need" and adds 1 to `owner-prompts:` for each PR "that message covers", `.agents/skills/coordinate/SKILL.md` § 4 and `internal/bootstrap/templates.go:421`). Accept: the sentences in both copies.
- [ ] 2.2 Coordinator contract's `owner-prompts:` sentence carries the counting rule (premise: "add 1 to the `owner-prompts:` line of each pull request that message covers", `.agents/contracts/coordinator.md` and `templates.go:370`). Accept: the clause in both copies.
- [ ] 2.3 Developer contract covers writing `Approved-by:` on the owner's own words and the hand-off when the host refuses the edit (premise: the contract says nothing about `Approved-by:`, `.agents/contracts/developer.md` and `developerContract`, `internal/bootstrap/review.go:17`, kept equal by `internal/bootstrap/review_test.go`). Accept: the paragraph in both copies.
- [ ] 2.4 Shipped-text test in `internal/bootstrap/coordinator_test.go` citing the new scenario; mutation of each template string shown as `detection`. Harness spec delta applied, carrying the scenarios of #447 (PR #486) and #448 (PR #487) if they land first; `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
