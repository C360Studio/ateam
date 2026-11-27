# Tasks: status-label-route-443

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-27) sha256:66acd21338b3`).

## 2. Implementation

- [ ] 2.1 `.agents/contracts/coordinator.md:9` and `coordinatorContract` (`internal/bootstrap/templates.go:372`): add the needs-decision issue and, on a Q1 finding, the blocker-labelled issue to the judge's gates; one whose blocker is open or unnamed stays an owner gate without ending the run. Acceptance: byte parity in `TestInitShipsJudgeContract`.
- [ ] 2.2 `.agents/contracts/judge.md:5`, `:9` and `judgeContract` (`templates.go:382`, `:386`): label removal, the `Ruling:` line, quoted blocker states, never add a label, `gate-change-approved` and `class:flake` owner-only. Keep the pinned literals at `coordinator_test.go:96-103`.
- [ ] 2.3 `.agents/skills/coordinate/SKILL.md:30` and `templates.go:423`: the issue and its `queue-check` finding in the evidence list.
- [ ] 2.4 `docs/model-review.md:157`: both labels in the judge's list and a *Labels* bullet.
- [ ] 2.5 New literals in `TestInitShipsJudgeContract` (`coordinator_test.go:94`), each shown able to fail with `task mutate:check` on `templates.go`.
- [ ] 2.6 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`.
- [ ] 2.7 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
