# Tasks: shortcut-rustdoc-commas-396

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-23) sha256:a7c0fa07553e`).

## 2. Implementation

- [x] 2.1 Accept `///`, `//!` and a nested `//` in `shortcutPattern`; split the limit at the first top-level comma with the first-comma fallback.
- [x] 2.2 Tests for each form, the bracketed comma, and the unbalanced fallback; a fuzz target against a reference model; mutation checks.
- [x] 2.3 Update the `docs/adoption.md` shortcut-syntax row and the harness scenario.
- [x] 2.4 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
