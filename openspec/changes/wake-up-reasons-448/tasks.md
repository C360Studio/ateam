# Tasks: wake-up-reasons-448

## 1. Approval

- [ ] 1.1 Owner approves the proposal.
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Coordinate step 2 names six reasons and the duplicate rule (premise: the launch prompt names `ready`, `blocked`, `question` and is sent only after a handoff, and the step lets the next unit start "once the draft PR claims the issue", `.agents/skills/coordinate/SKILL.md` § 2 and `internal/bootstrap/templates.go:413`). Accept: the prompt and the sentences in both copies.
- [ ] 2.2 Coordinate step 4 reads a question from the PR comment or `design.md` section the handoff names (premise: step 4 reads only "the PR comment its handoff names", `internal/bootstrap/templates.go:421`). Accept: the clause in both copies.
- [ ] 2.3 Developer contract covers design-track questions and `holding` (premise: its question paragraph names only a PR comment, `.agents/contracts/developer.md` and `developerContract`, `internal/bootstrap/review.go:17`, kept equal by `internal/bootstrap/review_test.go:152`). Accept: the sentences in both copies.
- [ ] 2.4 Shipped-text test in `internal/bootstrap/coordinator_test.go` citing the new scenario; mutation of each template string shown as `detection`. Harness spec delta applied, carrying the scenario of #447 (PR #486) if it lands first; `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
