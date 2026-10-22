## 1. Implementation

- [x] 1.1 Label a missing disk binary `absent` and a failed or malformed `--version` read `unreadable` in `ReadDiskBinaryBuild` (#121).
- [x] 1.2 Table test `TestDiskBuildFailureIsNotReportedAsTheRunningBuild` (no binary, non-zero exit, no parentheses) asserting the label, the HEAD stale target, and the pickup header; `task mutate:check -- -pkg ./internal/state -test TestDiskBuildFailureIsNotReportedAsTheRunningBuild -file internal/state/state.go` with the pre-fix file and with only the no-parentheses branch reverted, both `detection` at sha256 `bbf11b29` (#121).
