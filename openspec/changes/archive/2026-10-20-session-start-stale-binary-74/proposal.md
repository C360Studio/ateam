# Proposal: Warn at Session Start When Binary Is Behind HEAD (`#74`)

## Why

When multiple agents merge to `main`, the `ateam` binary on `PATH` (`./bin/ateam`) drifts behind checkout `HEAD`. While `ateam doctor --check` reported build-commit drift and `[ateam]` printed `head=<sha>` and `build=<sha>` side by side in the `session-start` header, `session-start` did not state explicitly that the binary was behind `HEAD`.

## What Changes

- Factor the shared stale-binary and missing-provenance messages into `bootstrap.FormatStaleBinaryWarning` (`binary <build> is behind HEAD <head>; run task build` and `binary has no build provenance; build with task build`) and use them in both `bootstrap.CheckBuildProvenance` (`ateam doctor`) and `session-start` (`internal/hook`).
- Compare `state.BuildRevision()` against `state.ReadFastGitHead` in `ateam hook --event session-start` without spawning a git subprocess so hook latency stays well within the `<5ms` (`5000 us`) budget.
- Add scenario `Stale or missing binary build provenance warning at session start` to `openspec/specs/harness/spec.md` and unit/mutation coverage in `internal/hook/hook_test.go` and `internal/bootstrap/parity_test.go`.
