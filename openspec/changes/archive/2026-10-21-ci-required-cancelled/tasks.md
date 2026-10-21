# Tasks: `ci-required-cancelled`

## 1. Implementation & Verification

- [x] 1.1 Change `concurrency.cancel-in-progress` in `.github/workflows/ci.yml` so a `ready_for_review` run does not cancel the in-flight run for the same head (#87).
- [x] 1.2 Pin the expression in `ciWorkflowViolations` with the fixture block and two `TestCIWorkflowPinnedSensitivity` cases; prove detection by mutation (`cp` + SHA-256 on `ci.yml` for `TestCIWorkflowPinned`, `ateam mutate` overlay on the checker for the sensitivity cases).
- [x] 1.3 Update the "Pinned CI workflow and Required aggregate" scenario in `openspec/specs/harness/spec.md`; `task spec:check`, `task test:spec`, `task lint`, and `go test ./internal/bootstrap` green.
