# Tasks: proposal-decisions-398

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-24) sha256:989b95b1c226`); rulings D1 A, D2 A, thresholds 5/500/60 in the #398 comment of 2026-11-24.

## 2. Implementation

- [x] 2.1 Decisions-section check (premise `internal/state/openspec.go:64,344,585-606`): parse the heading and bullets next to `ParseProposal`; fail with `design.md`, warn without; warn on a bullet under 5 plain words. Accept: table test with literal proposals for each scenario.
- [x] 2.2 Delta length check (premise `internal/state/openspec.go:655,691-695`): treat `ADDED`/`MODIFIED Requirements` as requirement sections, walk `openspec/changes/*/specs/**/spec.md`, warn past 500 characters and 60 words. Accept: 500/501-character and 60/61-word literal fixtures.
- [x] 2.3 Guidance (premise `.agents/protocol.md:30`, `internal/bootstrap/templates.go:294,336-348,349,361-385`): protocol § 2 step 1, design skill, architect contract, `rules: proposal:` in `openspec/config.yaml`, each in its repository copy and template. Accept: bootstrap test asserts each sentence in generated and repository files.
- [x] 2.4 Mutation check per rule (`task mutate:check`); `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
