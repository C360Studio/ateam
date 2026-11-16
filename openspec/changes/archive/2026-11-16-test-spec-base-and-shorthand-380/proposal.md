# Change Proposal: `test-spec-base-and-shorthand-380`

Approved-by: Coby Leuschke (2026-11-16) sha256:4e2125f07652

## Why

1. `ateam test-spec -base <ref>` still fails on tests the diff never touched when those tests carry a `// spec:` line that is not in ateam's format (`internal/testspec/check.go:180-185`). In SemEngine at `c6ef1d84`, an empty diff (`ateam test-spec -base HEAD`) reports 257 `malformed spec citation` lines across 61 untouched test files, blocking brownfield adoption of the `-base` gate (#380).
2. SemEngine and SemTeams have hundreds of legacy citations written in the short form `// spec: <capability> / <requirement>` (for example `// spec: component-lifecycle / The shell admits one Start after Initialize`), which names the same requirement as `openspec/specs/<capability>/spec.md#Requirement: <requirement>`. Without accepting that form, any PR that edits a legacy test must also rewrite its citation line.

The owner ruled on 2026-11-16 (#380) to fix both in `v0.4.2`.

## What Changes

- Under `--base <ref>`, `malformed spec citation` on a `// spec:` line is reported only when the declaration's span meets a changed line (`changed == nil || changed[d.File].touches(d)`), matching `missing spec citation`. Resolved citations that no longer match a requirement heading (`requirement resolves 0 times`) still fail across the whole tree so renaming a requirement catches untouched callers.
- `resolve` accepts the short form `<capability> / <requirement>` (where `<capability>` is a single path segment `[a-z0-9][a-z0-9_-]*` with no `#` or `/`, separated by ` / ` from a non-empty requirement title) by checking `openspec/specs/<capability>/spec.md` first and, if no `### Requirement: <requirement>` heading is found there, checking non-archived `openspec/changes/*/specs/<capability>/spec.md`. Exactly 1 match across the checked scope succeeds; 0 or >1 fails with `requirement resolves <n> times: <ref>`.
- `.agents/protocol.md`, `internal/bootstrap/templates.go`, `docs/brownfield.md`, and `openspec/specs/test-attribution/spec.md` document the short form and the `-base` scope for malformed citations.

## Track

`quick`. It modifies two existing Requirements (*Safe requirement resolution* and *Diff-scoped attribution*) in `test-attribution`, adds no new Requirement or CLI flag, and touches no gate file.

## Non-goals

- Scoping dangling citations (`requirement resolves 0 times` on a syntactically valid citation) or source parse errors to `--base`: renaming a spec heading must still flag untouched tests that cite the old heading.
- Arbitrary custom citation templates or per-repo regex configuration.

## Definition of Done

- `TestGoAttribution`, `TestReferenceRefusals`, and `TestBaseScope` cover the `<capability> / <requirement>` shorthand (canonical, in-flight fallback, archived ignored, missing/duplicate failure) and untouched malformed citations passing under `--base` while failing when the test is edited or when `--base` is omitted.
- `task mutate:check` reports `detection` for both new branches; `task verify`, `ateam doctor --check`, and `ateam bench --selftest` pass; change is archived before the PR is marked ready.
