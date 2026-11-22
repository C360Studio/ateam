# Proposal: debt-scan-rust-python-383
Approved-by: Coby Leuschke (2026-11-22) sha256:8cb33b506e0a

## Track

`quick` (one capability, `harness`; extends an existing Requirement with one scenario, no new Requirement, no exported surface).

## Why

`ateam state debt` skips `.rs` and `.py`, so Rust and Python adopters (#382) see no `// shortcut:` / `# shortcut:` debt. Cargo writes `target/` per crate, so a nested `firmware/<crate>/target/` must be skipped too.

## What Changes

1. Add `.rs` and `.py` to `sourceExts` in `internal/state/shortcut.go`; both comment leaders are already accepted.
2. Add `target`, `.venv`, and `__pycache__` to `skippedDirs`, which match by directory name at any depth.
3. Add a scenario to *Inline shortcut inventory* and update the `docs/adoption.md` limits row.

## Scope & Non-Goals

- **In-bounds:** `internal/state/shortcut.go`, `internal/state/state_test.go`, `openspec/specs/harness/spec.md`, `docs/adoption.md`.
- **Non-goals:** pytest or Rust test-to-spec attribution (#384), skipping Rust/Python test files, mutation for Rust (#385).

## Definition of Done

- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; `task mutate:check` detects removal of the `target` skip and of `.rs`.
- Change archived before the PR is marked ready.
