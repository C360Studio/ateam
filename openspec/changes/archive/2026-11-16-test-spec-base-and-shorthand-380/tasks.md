# Tasks: test-spec-base-and-shorthand-380

## 1. Approval

- [x] 1.1 Record owner rulings from #380 and approve proposal (`Approved-by: Coby Leuschke (2026-11-16) sha256:4e2125f07652`).

## 2. Implementation & Verification

- [x] 2.1 Update `internal/testspec/check.go` so `resolve` accepts `<capability> / <requirement>` against canonical `openspec/specs/<capability>/spec.md` first and non-archived `openspec/changes/*/specs/<capability>/spec.md` second, and `Check` scopes per-test malformed citation diagnostics (`malformed spec citation`, `write //aip:verifies without a space`, `aip:verifies names no scenario IDs`) to changed tests under `--base`.
- [x] 2.2 Add regression tests in `internal/testspec/check_test.go` and `internal/testspec/diff_test.go` covering canonical and in-flight `<capability> / <requirement>` resolution, archived/duplicate/missing refusal, and `--base` suppression of malformed citations on untouched tests vs failure on touched tests.
- [x] 2.3 Update `.agents/protocol.md`, `internal/bootstrap/templates.go`, `docs/brownfield.md`, and `openspec/specs/test-attribution/spec.md` to match the spec delta.
- [x] 2.4 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`, and non-destructive `task mutate:check` (`detection` on `TestBaseScope` and `TestCapabilityShorthand`), then archive this change.
