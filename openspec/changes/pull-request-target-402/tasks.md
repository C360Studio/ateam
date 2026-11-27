# Tasks: pull-request-target-402

## 1. Approval

- [ ] 1.1 Owner approves the proposal and design and rules on Open Questions 1 and 2.
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Gate (premise `internal/review/gate.go:286-318`, `:49`, `:153-156`; a gate file): `pull_request_target` is a pull-request run on GitHub Actions, and on Gitea Actions exits 2 naming the context reason. Accept: rows in `internal/bootstrap/mergecheck_test.go` beside `push run` (`:378`) for `pull_request_target` pass and fail on GitHub, refusal on Gitea, and an unknown event; `task mutate:check` detection on dropping the new arm.
- [ ] 2.2 Live run (premise `internal/review/gate_github.go:545-590`, D2): one `pull_request_target` run in the agreed private repository. Accept: a #402 comment with the check run's `head_sha`, whether the ruleset counted it, and the `--awaiting` output after a pass.
- [ ] 2.3 Guide (premise `docs/adoption.md:135`, D3): a trigger row saying private/internal → `pull_request_target`, public → `pull_request` plus `Gate files`, never check out PR code under the target trigger, and the rerun lookup if 2.2 shows runs under the base branch. Accept: `markdownlint` clean and the row matches 2.2's comment.
- [ ] 2.4 Spec (premise `openspec/specs/harness/spec.md:501`): the ADDED Requirement below. Accept: `task spec:check`.
- [ ] 2.5 The owner applies `gate-change-approved` at the head. Run `task verify`, `ateam doctor --check` and `ateam bench --selftest`, then archive this change.
