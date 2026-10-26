# Change Proposal: `aip-verifies-129`

## Why

AIP attributes tests to specs with a `//aip:verifies <scenario IDs>` doc-comment directive (its TRACE-003), not a `// spec: <path>#Requirement: <name>` citation. `ateam test-spec` accepts only the `spec:` form (`internal/testspec/check.go`), so every AIP test reports `missing spec citation` and `task test:spec` cannot run in an AIP checkout (#129, AIP ask 2).

## What Changes

- An attached comment whose raw text starts with `//aip:verifies` is a citation. It carries one or more space-separated scenario IDs; each ID resolves when a `#### Scenario: <ID> ` header (outside code fences) exists in a canonical `openspec/specs/*/spec.md` or an in-flight `openspec/changes/*/specs/*/spec.md`, never under `archive/`. Spec files keep the existing repository-escape and UTF-8 checks.
- The spaced form `// aip:verifies`, a directive with no IDs, and an unknown ID each produce a diagnostic. The `spec:` form is unchanged; a test may carry either or both.

## Non-goals

- AIP's one-capability-per-test rule (TRACE-004), scenario coverage, and the `e2e` build-tag scan; those belong to `aip trace check`.
- Rejecting an ID that appears more than once across canonical and in-flight specs: a change's delta legitimately repeats a canonical scenario.
