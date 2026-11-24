# Tasks: gemini-neon-hydrogen-405

## 1. Approval

- [x] 1.1 Owner approves proposal.md and design.md together (`Approved-by: Coby Leuschke (2026-11-24) sha256:1e9db993a0a3`; D1 B, D2 B, D3 drop).

## 2. Implementation

- [x] 2.1 Edit tiers and the Gemini `high` band (premise: `.agents/review-policy.json:4-6,11`). Check: `ateam doctor --check` and `ateam review compare` with the proposal's DoD pair.
- [x] 2.2 Update the committed-policy copy (premise: `internal/review/compare_test.go:18`). Check: `go test ./internal/review -run TestCommittedPolicyTiers` passes; a mutation that keeps `gemini-3.1-pro` in the copy detects.
- [x] 2.3 Amend ADR 0002 Status, § 1 tables, JSON and the effort row (premise: `docs/adr/0002-model-tiers-and-effort-bands.md:3,24,36-38,46,58-65`).
- [x] 2.4 Add the agy suffix and Orchestrator note (premise: `docs/model-review.md:53`).

## 3. Close

- [x] 3.1 `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
