## 1. Implementation

- [ ] 1.2 `cmd/ateam/main.go`: set `hookLatencyMaxUs` to `100000`. `TestHookLatencyFindings` boundary rows move to 100000 and 100001.
- [ ] 1.3 `internal/hook/hook_test.go`: `TestHookLatencyBudget` fails above 100ms.
- [ ] 1.4 Rule and doc text says 100ms in `AGENTS.md`, `.agents/rules-index.md`, `.agents/protocol.md`, `internal/bootstrap/templates.go`, README, `docs/adoption.md`, `docs/daily-use.md`, and `docs/hook-flow.md`. Apply the spec delta to `openspec/specs/harness/spec.md`.
- [ ] 1.5 `task mutate:check`: a mutant reverting `hookLatencyMaxUs` to `25000` reports `detection` against `TestHookLatencyFindings`.
- [ ] 1.6 `task verify` passes on a clean tree, and `ateam doctor --check` passes.
