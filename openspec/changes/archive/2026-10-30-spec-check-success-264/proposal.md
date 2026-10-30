# Change Proposal: `spec-check-success-264`

Approved-by: Coby Leuschke (2026-10-30)

## Why

`ateam state spec-check` prints an `openspec:` line only when strict validation is skipped (`internal/state/openspec.go:526`). When `openspec validate --all --strict` runs and passes, the output is just `holds: ok (...)`, so an adopter cannot tell a real pass from a silent skip, nor which binary or version ran (#264, asked for in #231).

## What Changes

1. After a passing strict run, `CheckOpenSpec` writes `openspec: strict ok (<resolved path>, <version>, <n> changes)`. The version is the first line of `openspec --version` (one extra exec under the same timeout), or `version unknown` when that prints nothing or fails. `<n>` counts the in-flight changes the queue scan reads. A failed run and the skip notice are unchanged.
2. `docs/brownfield.md` § OpenSpec tracks names the success line beside the skip notice.
3. Spec: a scenario added to the harness Requirement "OpenSpec queue and stray Hold detection".

## Track

`quick`. One function in `internal/state` and one doc. It adds a scenario but no Requirement, adds no exported surface, and touches no boot path, durability behavior, or gate file. The owner approved the issue's proposed fix in Enjoy on 2026-10-30.

## Non-goals

- Changing the skip notice or the failure output.
- Caching the version across runs; one exec per check is enough.

## Definition of Done

- `TestCheckOpenSpecFindsValidatorOnPATH` asserts the success line names the PATH binary with its version, names the local binary with `version unknown`, and is absent when the validator is skipped or fails.
- `task mutate:check` reports `detection` when the success line is dropped.
- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
