# Tasks: whole-review-marker-399

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-24) sha256:10b7a17c4a6b`); rulings D1 A, D2 A, D4 A in the #399 comment of 2026-11-24.

## 2. Implementation

- [x] 2.1 Counts (premise `internal/state/state.go:555-570,2789-2830`): newest reachable `review/whole/` tag, first-parent count and `--shortstat` to the remote tip, cached in `GitStatus`. Accept: temp-repo tests for none, two commits +3/-1, newest of two tags.
- [x] 2.2 Display (premise `internal/state/state.go:3074`, `internal/hook/hook.go:121,152`): `since-review=` on the summary line and in status JSON; hook path unchanged. Accept: literal summary-line assertions.
- [x] 2.3 Command (premise `cmd/ateam/review.go:21-43`, `internal/state/prune.go:249`): `ateam review whole --at --note --no-push`; refuse off-branch commits. Accept: tag created, `origin` untouched with `--no-push`, refusal exit code.
- [x] 2.4 Guidance (premise `.agents/protocol.md:45`, `internal/bootstrap/templates.go:653`, `internal/bootstrap/reviewrecord.go:29`): Close-step sentence in repository copy and both templates; README and `docs/daily-use.md`. Accept: bootstrap wording test.
- [x] 2.5 Mutation check per rule; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
