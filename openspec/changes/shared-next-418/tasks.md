# Tasks: shared-next-418

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-26) sha256:79a60d9e4537`), written by claude on the owner's instruction; rulings D2 A, D3 A, D5 A in the #418 comment of 2026-11-26.

## 2. Implementation

- [x] 2.1 Sync (premise `internal/state/forge.go:236-241,655,660-672,400-427,753-815`): add the milestone description to the issue payload, the PR `body` to `prFields`, and `DependsOn`/`Claims` to `BoardCard` (`internal/state/state.go:575-600`). Record a comments-unread flag when the GitHub open-issue list falls back without comments. Accept: fixture JSON for both forges fills the fields, and the fallback sets the flag.
- [x] 2.2 Compute (new `internal/state/next.go`): parse `Order:`/`After:`/`Depends on:`, choose the earliest milestone, and walk its order, recording a skip reason per issue plus the unplaced issues. Accept: table tests for every reason in D2–D5, a shuffle property test, and a mutation check per rule.
- [x] 2.3 Display (premise `internal/state/state.go:3057-3083`, `cmd/ateam/main.go:133-170`): `next=` on the summary line, `ateam state next`, and the status JSON field. The hook path is unchanged. Accept: literal summary-line and command-output assertions.
- [x] 2.4 Skills (premise `.agents/skills/pickup/SKILL.md:36`, `.agents/skills/handoff/SKILL.md:31`, `internal/bootstrap/templates.go`): pickup reads `ateam state next`, and the handoff Goal carries no order. Covers the repository copy and the templates, plus `docs/daily-use.md`. Accept: bootstrap wording test and `ateam doctor --check`.
- [ ] 2.5 Add `After:` lines to this repository's open milestones (the owner confirms the text). Then `task verify`, `ateam doctor --check` and `ateam bench --selftest`, and archive this change.
