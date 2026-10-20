# Tasks: Warn at Session Start When Binary Is Behind HEAD (`#74`)

## 1. Implementation & Verification

- [x] 1.1 Share `FormatStaleBinaryWarning` between `bootstrap.CheckBuildProvenance` and `internal/hook` (`binary <x> is behind HEAD <y>; run task build` and `binary has no build provenance; build with task build`).
- [x] 1.2 Emit the stale or missing build provenance line once on `session-start` using `ReadFastGitHead` and `state.BuildRevision()` with zero subprocesses.
- [x] 1.3 Add `TestSessionStartWarnsWhenBinaryBehindHead` in `internal/hook/hook_test.go`, prove failure via `task mutate:check`, and verify `ateam doctor --check` hook latency and `task verify`.
