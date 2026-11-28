# Tasks: developer-handoff-fields-447

## 1. Approval

- [ ] 1.1 Owner approves the proposal.
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Developer contract carries the four handoff and PR-body rules (premise: `.agents/contracts/developer.md` says nothing on owner-owed actions, the track, gate-file claims or the `owner-prompts:` line; its copy is `developerContract`, `internal/bootstrap/review.go:17`, kept equal by `internal/bootstrap/review_test.go:152`; the coordinator writes the line, `internal/bootstrap/templates.go:368`). Accept: the paragraph in both copies.
- [ ] 2.2 Coordinate step 4's owner message lists owed actions and repeats each proposal's track (premise: step 4 lists the PR or issue, the question and the proposal digest only, `.agents/skills/coordinate/SKILL.md` § 4 and `internal/bootstrap/templates.go:421`). Accept: the sentence in both copies.
- [ ] 2.3 Shipped-text test in `internal/bootstrap/coordinator_test.go` citing the new scenario; mutation of each template string shown as `detection`. Harness spec delta applied, carrying #446's scenario if PR #484 lands first; `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
