# Tasks: rust-adopter-guidance-386

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-22) sha256:2b1bbfe0dcc0`).

## 2. Implementation

- [ ] 2.1 Add the Rust section to `docs/adoption.md`, fix its `ateam test-spec` and `ateam mutate` rows, and point `docs/brownfield.md` at it.
- [ ] 2.2 Reword the Go-only spots in `internal/bootstrap/templates.go` (rules rows, protocol mutation and generated-check steps, limits sentence) and name `cargo-mutants` in `.agents/protocol.md`.
- [ ] 2.3 Add the harness scenario; check a fresh `ateam init` fixture with `ateam doctor --check`.
- [ ] 2.4 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
