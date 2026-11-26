# Tasks: hold-my-beer-424

## 1. Approval

- [ ] 1.1 Owner approves the proposal and design, with rulings on D1–D5. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Launch (dogfood, #417): a coordinator session in default mode, in its own `<agent>/coordinator` worktree, starts the developer run for this change and moves its PR. The PR body carries `owner-prompts:`, and the count is commented on #417. Accept: the merged PR body and the #417 comment.
- [ ] 2.2 Policy (premise `internal/review/policy.go:13-42`): `HoldMyBeer bool` with json `hold_my_beer`, absent meaning false. Accept: parse tests for absent, true and false.
- [ ] 2.3 Record (premise `internal/state/approval.go:38-101`, `internal/state/openspec.go:688-710`): parse an unfenced `Judged-by:`; `ApprovalDigest` drops both lines; add a judged approval state. `CheckOpenSpec` fails a filled `Judged-by:` while the opt-in is off and treats it like `Approved-by:` for staleness while it is on; a filled `Approved-by:` wins. Accept: a table test over owner-only, judge-only, both, stale, unbound and fenced lines with the opt-in on and off, using literal digests; a digest-invariance test; `task mutate:check` detection on the opt-in branch.
- [ ] 2.4 Display (premise `internal/state/commitstatus.go:114-129`, `internal/state/openspec.go:555-612`): `ateam/proposal` reads `judged by <agent> (<model>, <effort>)`, and `spec-queue` labels a judged change. Accept: literal status and queue-line assertions.
- [ ] 2.5 Merge-check note (premise `internal/review/gate.go:409-425`; a gate file): a judged change's note says `judged by <agent>, not owner-reviewed`, and the exit status is unchanged. Accept: a gate test with a fixture proposal.
- [ ] 2.6 Roles (premise `docs/adr/0003-roles-and-routes.md:30,63-67`, `.agents/contracts/coordinator.md`, `.agents/skills/coordinate/SKILL.md`, `internal/bootstrap/templates.go`): `.agents/contracts/judge.md` written by init; a hold-my-beer section in the contract and skill covering D3 compare, D4, D6 and the ruling-5 stops; the ADR 0003 amendment; `docs/model-review.md` and `docs/daily-use.md`. Accept: the bootstrap test extended to the judge contract, with literal-sentence assertions, and `ateam doctor --check`.
- [ ] 2.7 The owner applies `gate-change-approved` at the head. Run `task verify`, `ateam doctor --check` and `ateam bench --selftest`, then archive this change.
