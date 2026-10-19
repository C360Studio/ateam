# Tasks: Native Antigravity (.agents/hooks.json and injectSteps) support

implemented-by: gemini

## 1. Hook handler and CLI

- [x] 1.1 Support `conversationId`, `PreInvocation` normalization, `injectSteps`, and `Stop` (`decision: "allow"`) for `--agent agy` in `internal/hook/hook.go` and `cmd/ateam/main.go`.
- [x] 1.2 Add unit tests in `internal/hook/hook_test.go` using an independent oracle and prove failure detection with `task mutate:check`.

## 2. Bootstrap, doctor parity, and verification

- [x] 2.1 Add `.agents/hooks.json` to `internal/bootstrap/templates.go`, this repository, and `checkHookConfigs` in `internal/bootstrap/bootstrap.go`, with sensitivity coverage in `internal/bootstrap/parity_test.go`.
- [x] 2.2 Update `.agents/README.md`, `AGENTS.md`, `.agents/rules-index.md`, and `openspec/specs/harness/spec.md`, and verify with `ateam doctor --check` and `task verify`.
