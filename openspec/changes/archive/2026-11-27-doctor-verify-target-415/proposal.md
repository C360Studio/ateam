# Proposal: doctor-verify-target-415
Approved-by: Coby Leuschke (2026-11-27) sha256:e469985b79f1

## Track

`design`: adds a spec Requirement (`Doctor checks the verify target`) and changes what `ateam doctor --check` returns in an adopting repository. See `design.md`.

## Decisions you are making by approving

- When a repository has no `Taskfile.yml`, or has one with no `verify` task, `ateam doctor --check` fails and names what is missing. An adopter whose CI runs the check goes red on upgrade until it adds the task; today their reviews break instead, one at a time.
- Doctor asks the Task tool itself for the list of tasks, so tasks pulled in from included files count, and reads the file directly when Task is not installed. A `verify` that exists only as an alias of another task does not count, the same as for `ateam task verify` today.
- Nothing new to configure: no setting names a different verify command.

## Why

The adopter contract asks for a `task verify` target (`docs/daily-use.md:3`, `docs/adoption.md:112`), and the shipped preflight and review-pr skills run it by name, yet `ateam doctor --check` never looks for it: the only `task verify` check reads the preflight skill's text (`internal/bootstrap/bootstrap.go:446`). In semsource (70c42d1), `doctor --check` passed while `task verify` failed with `Task "verify" does not exist`; four reviews hit it, three substituted `task check`, and one asked for changes on that alone (#415).

## What Changes

1. `ateam doctor` adds one violation when the target is missing: `Taskfile.yml: missing; the shipped skills and protocol run task verify` or `Taskfile.yml: no verify target; the shipped skills and protocol run task verify`.
2. Target discovery reuses `taskrunner.ListTargets`, the lookup `ateam task` already uses.
3. Harness spec gains the Requirement with three scenarios; `docs/adoption.md` names the check in the `ateam doctor` row.

## Scope & Non-Goals

- **In-bounds:** `internal/bootstrap` (one new check), `cmd/ateam/main.go` (`runDoctor` calls it), their tests, `openspec/specs/harness/spec.md`, `docs/adoption.md`.
- **Non-goals:** `CheckParity` and `ateam init` (init still writes no Taskfile, and its parity tests stay green); a configurable verify command; Task aliases; `Taskfile.yaml` or other Task file names; checking what `verify` runs; every gate file.

## Definition of Done

- Tests on temp repositories: no `Taskfile.yml` → the missing violation; a Taskfile with `check` and no `verify` (the semsource shape) → the no-target violation and a non-zero exit; a Taskfile with `verify` → no such violation; a `verify` reached only through a flattened `includes:` passes when Task is installed. A mutation check detects dropping the target-name comparison.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` green in this checkout; the change is archived before the PR is marked ready.
