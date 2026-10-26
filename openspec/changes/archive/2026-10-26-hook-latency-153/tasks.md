## 1. Implementation

- [x] 1.1 `cmd/ateam/main.go`: `hookLatencyBudgetUs` 5000 to 10000, finding logic extracted to `hookLatencyFindings`; rule text updated in AGENTS.md, `.agents/rules-index.md`, `.agents/protocol.md`, `internal/bootstrap/templates.go`, README, `docs/adoption.md`, and comments (#153).
- [x] 1.2 `TestHookLatencyFindings` in `cmd/ateam/main_test.go`; `TestHookLatencyBudget` logs above 10ms and fails above 25ms.
- [x] 1.3 Spec delta: MODIFIED requirement renamed to `Lifecycle hook normalization under 10ms`, applied to `openspec/specs/harness/spec.md`; `// spec:` citations updated.
- [x] 1.4 `task mutate:check` mutant `hookLatencyBudgetUs = 10000` reverted to `5000` (mutant sha256 `cc227473...db9603`) against `TestHookLatencyFindings`, `-expect-text 'hookLatencyFindings(6000)'`: `detection`; `cmd/ateam/main.go` unchanged afterwards (sha256 `2848955b...0309c6` before and after).
