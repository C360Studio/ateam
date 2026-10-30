# Change Proposal: `test-spec-base-237`

Approved-by: Coby Leuschke (2026-10-30)

## Why

`ateam test-spec` checks every Go and Vitest test in the repository and takes no arguments (`internal/testspec/check.go:48`, `cmd/ateam/main.go:667-671`). A brownfield adopter's first run fails on every existing test: SemTeams `main` reports 4,732 tests with `missing spec citation`, so it deferred the rule (#237). The only choices today are a full backfill no reviewer can check, or no gate at all.

## What Changes

- `ateam test-spec --base <ref>` reports `missing spec citation` only for tests whose declaration or attached comment lines are added or changed between the merge base of `<ref>` and the working tree (committed, uncommitted, and untracked). Without `--base` the command behaves exactly as today.
- An unresolvable `<ref>` (unknown ref, shallow clone without the merge base, not a git repository) exits 2 with a fetch hint. It never falls back to a full or an empty scope.
- `test-spec --help`, protocol § Test-to-spec attribution, and `docs/brownfield.md` say a brownfield repository starts with `--base` in its PR gate (with `fetch-depth: 0`) and a full run once its backfill is done.
- The `test-attribution` spec gains a Requirement for the diff scope.

## Track

`design`: it adds a spec Requirement and adopter-visible CLI surface. See `design.md`.

## Non-goals

- The committed baseline (`--write-baseline`, the issue's second option): deferred until SemTeams asks for it after trying `--base`.
- This repository's own `task test:spec` stays a full run; every test here is cited.
- No change to `ateam review gate` or any gate file; adopters wire `--base` into their own CI.
- No migration hint for SemTeams' older `// spec: <capability> / <title>` form; it stays `malformed spec citation`.

## Definition of Done

- A fixture git repository with uncited legacy tests (Go and Vitest) passes `ateam test-spec --base main` after a commit adds one cited test, and fails naming the test when the new test is uncited, or when an uncited legacy test's body is edited.
- An unknown or shallow `--base` exits 2; no `--base` keeps today's diagnostics byte-for-byte.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
