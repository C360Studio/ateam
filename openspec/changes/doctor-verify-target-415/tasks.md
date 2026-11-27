# Tasks: doctor-verify-target-415

## 1. Approval

- [ ] 1.1 Owner approves the proposal and design.
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `bootstrap.CheckVerifyTarget(ctx, root) []string` (premise: `taskrunner.ListTargets` at `internal/taskrunner/taskrunner.go:25` returns an error naming `Taskfile.yml` when the file is absent (`:28-30`), asks `task --list-all --json` (`:44`) and falls back to the top-level scan (`:70-71`)). Returns the missing-file violation when `ListTargets` cannot read the file, the no-target violation when no listed name is `verify`, nothing otherwise. Accept: table test over no file, `check` only, `verify` present, `verify` in a flattened include (skipped when `task` is not on `PATH`); mutation check on the name comparison.
- [ ] 2.2 `runDoctor` appends it to `violations` after `CheckParity` (premise: `cmd/ateam/main.go:656`, verdict at `:709`). Accept: a `cmd/ateam` test runs doctor on the semsource shape and sees the violation and exit 1.
- [ ] 2.3 Harness spec delta applied; `docs/adoption.md:20` names the check; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
