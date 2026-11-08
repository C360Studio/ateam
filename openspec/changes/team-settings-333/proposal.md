# Change Proposal: `team-settings-333`

Approved-by:

## Why

ateam has no committed home for settings a team chooses. The only committed config, `.agents/review-policy.json`, is a gate file with strict decoding and required model tiers. Team preferences have scattered into env vars (`ATEAM_COMMIT_STATUSES`), the gitignored `.ateam/state.json` cache (the forge list), or hardcoded text. Generated instructions always say `gh pr merge --squash --auto` (`internal/bootstrap/templates.go:637`, `:643`), and `main`/`master` is hardcoded across `internal/state`. That makes these claims wrong for a team that uses merge commits or another default branch (#333).

## What Changes

- New committed, non-gate `.agents/ateam.json` (strict, `version: 1`). If it is missing, behaviour stays as it is today and bootstrap output is byte-identical.
- `merge_method` (`squash` default, `merge`, `rebase`) renders the GitHub `gh pr merge` flag, the Gitea `do` value, and the Close wording.
- `default_branch` replaces the hardcoded `main`/`master` and falls back to the detected remote default.
- `commit_statuses` and `forges` move in from `ATEAM_COMMIT_STATUSES` and the state cache. The env var stays as a documented per-machine override.
- `ateam doctor` validates the file.
- One new rule (AGENTS.md plus `.agents/rules-index.md`): a setting a team commits lives in `.agents/ateam.json` unless it can weaken a merge gate. `ATEAM_*` env vars carry only per-machine, per-session, secret or test values, or a documented override. `.ateam/` is runtime cache only.

## Track

`design`: it adds exported surface reachable from adopting repos, adds a spec Requirement, and touches bootstrap and state. See `design.md`.

## Non-goals

- The `required_checks` rename (#336, a gate-file change).
- Making ateam's own limits configurable (AGENTS.md bytes, proposal/design lines, hook budget).
- Moving gate-weakening keys out of `review-policy.json` or the workflows.
- Reading the merge methods a forge allows (possible later as a doctor warning).

## Definition of Done

- With no `.agents/ateam.json`, `ateam doctor --check` parity and the bootstrap golden output are unchanged.
- With `merge_method: merge` / `rebase`, the generated Close text names `--merge` / `--rebase` and the matching Gitea `do`.
- With `default_branch: trunk`, merge detection and prune treat `trunk` as the default branch.
- Parser and precedence tests are proven by `task mutate:check` (`detection`).
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green. The change is archived before ready.
