# Tasks: debt-scan-rust-python-383

## 1. Approval

- [x] 1.1 Record owner approval for #383 (`Approved-by: Coby Leuschke (2026-11-22) sha256:8cb33b506e0a`).

## 2. Implementation & Verification

- [x] 2.1 Add `.rs` / `.py` to `sourceExts` and `target` / `.venv` / `__pycache__` to `skippedDirs` in `internal/state/shortcut.go`.
- [x] 2.2 Add `TestScanShortcutsRustPythonAndNestedBuildDirs` covering root and nested `target/`, `.venv/`, `__pycache__/`; mutation-check the `target` skip and `.rs` entry.
- [x] 2.3 Update `openspec/specs/harness/spec.md` and `docs/adoption.md`.
- [x] 2.4 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`, then archive this change.
