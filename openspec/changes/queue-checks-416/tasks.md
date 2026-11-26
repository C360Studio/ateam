# Tasks: queue-checks-416

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-26) sha256:3258493157fd`), written by claude on the owner's instruction; rulings D1 A, D3 A, D4 A, Q5 default 15 in the #416 comment of 2026-11-26.

## 2. Implementation

- [ ] 2.1 Milestones read (premise `internal/state/forge.go:651` GitHub, `:357` Gitea): one open-milestones call per forge, cached as title plus open and closed issue counts. tea fallback: unavailable. Accept: fixture tests on both forges; an unsuccessful call sets the queue result to unavailable, not zero.
- [ ] 2.2 Checks (new `internal/state/queuecheck.go`; reuse `parseDependsOn`, `NextResult.Unplaced`, `BoardCard.Decision` from `internal/state/next.go`, `forge.go:865-896`): rules Q1–Q6 from design D2. Accept: hit and miss table test per rule, plus a mutation check per rule.
- [ ] 2.3 Setting (premise `internal/state/settings.go:29-38`): `queue.max_milestone_issues`, default 15, validated by `ateam doctor --check` (a positive integer). Accept: settings test for absent, valid and invalid values.
- [ ] 2.4 Display (premise `internal/state/state.go:3057-3090`, `cmd/ateam/main.go:133-175,624`): `queue=` on the summary line, `ateam state queue-check`, status JSON, and the doctor advice line from the cache. Accept: literal output assertions; the hook path and `doctor --check` make no forge call.
- [ ] 2.5 Guidance: preflight skill, plus its template, names `ateam state queue-check` before a handoff. Also `docs/daily-use.md` and `docs/adoption.md` § Team settings. Accept: bootstrap wording test and `ateam doctor --check`.
- [ ] 2.6 `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
