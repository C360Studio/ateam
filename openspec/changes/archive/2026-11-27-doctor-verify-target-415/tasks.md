# Tasks: doctor-verify-target-415

## 1. Approval

- [x] 1.1 Owner approves the proposal and design.

## 2. Implementation

- [x] 2.1 `bootstrap.CheckVerifyTarget(ctx, root) []string` (premise: `taskrunner.ListTargets` at `internal/taskrunner/taskrunner.go:25` returns an error naming `Taskfile.yml` when the file is absent (`:28-30`), asks `task --list-all --json` (`:44`) and falls back to the top-level scan (`:70-71`)). Returns the missing-file violation when `ListTargets` cannot read the file, the no-target violation when no listed name is `verify`, nothing otherwise. Accept: table test over no file, `check` only, `verify` present, `verify` in a flattened include (skipped when `task` is not on `PATH`); mutation check on the name comparison.
- [x] 2.2 `runDoctor` appends it to `violations` after `CheckParity` (premise: `cmd/ateam/main.go:656`, verdict at `:709`). Accept: a `cmd/ateam` test runs doctor on the semsource shape and sees the violation and exit 1.
- [x] 2.3 Harness spec delta applied; `docs/adoption.md:20` names the check; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
- [x] 2.4 Review round 1 (codex REQUEST CHANGES at 66dec81, CI run 46023974608): Task 3.43.3, pinned in CI, lists `name` and no `task` key, so `listWithTask` (`internal/taskrunner/taskrunner.go:61-70`) returned empty names without falling back and doctor rejected this repository's `verify`. A listing with an empty task key now falls back to the file scan. Accept: `name only` case in `TestListTargetsFallsBackWhenTaskCLIFails`, a name-only fake in `TestCheckVerifyTarget`, the flattened-include test skips when Task lists no task key; mutation checks detect dropping the guard; CI Verify green.
