# Tasks: third-rejection-stop-411

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-27) sha256:dc517106367f`), written by claude on the owner's instruction in its developer conversation; rulings D2 A, D3 A, D4 A in the PR #438 comment of 2026-11-27.

## 2. Implementation

- [ ] 2.1 Cache (premise `internal/state/forge.go:277-285`, `:308-315`; `internal/state/state.go:597-603`): beside `Review`, cache the rejecting records sync read (sha, agent, model, effort, link) and the count named by the newest unfenced `review-rounds-continue: <n>` line. Confirm first that both forges hand sync every comment: GitHub `gh pr view --json comments` (`forge.go:757`) and Gitea `fetchGiteaIssueComments` (`forge.go:412`, one page). Accept: table test over comment lists (oracle: hand-counted literals), including a fenced line and a record with no verdict.
- [ ] 2.2 Refusal (premise `cmd/ateam/reviewlaunch.go:151`, `:214-227`, `:321-332`): before the `--reviewer-session` branch and any launch, exit 2 when count ≥ 3 and the continue count ≠ count; stderr lists the rounds and the line to post. Accept: `reviewlaunch_test.go` cases from the Definition of Done, each with a `task mutate:check` detection.
- [ ] 2.3 Prose (premise `.agents/contracts/coordinator.md:7`, `:9`; `.agents/skills/coordinate/SKILL.md:22`, `:30`; `.agents/contracts/judge.md:9`; templates `internal/bootstrap/templates.go:370`, `:372`, `:386`, `:415`, `:423`): the coordinator relays the refusal's summary in its owner message; only the owner, or a judge the coordinator started, writes the continue line. Accept: `coordinator_test.go` asserts the new phrases; parity green.
- [ ] 2.4 Spec: ADDED Requirement `Owner decision after the third rejecting review` in `openspec/specs/model-review/spec.md`. Accept: `task spec:check`, `task test:spec`.
- [ ] 2.5 `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change per the Close rule live at merge.
