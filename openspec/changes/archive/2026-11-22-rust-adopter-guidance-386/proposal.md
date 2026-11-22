# Proposal: rust-adopter-guidance-386
Approved-by: Coby Leuschke (2026-11-22) sha256:2b1bbfe0dcc0

## Track

`quick`: docs and generated instruction wording, one MODIFIED Requirement (*Accurate lab positioning and adoption boundaries*, `harness`) gaining a scenario; no new Requirement, gate file, or exported surface. Closes #386 and #385.

## Why

#383 and #384 made `state debt` and `test-spec` handle Rust, but a Rust adopter still has no guide, `docs/adoption.md` still says Rust has no mutation recipe and that `test-spec` is Go/Vitest-only, and generated text names Go-only tools (`gofmt`/`go vet`/`revive`, `testing.F`, `go test -overlay`). The bbstack adoption (#382) needed hand edits for exactly these.

## What Changes

1. `docs/adoption.md` gains a **Rust repositories** section: what works unchanged; debt scan and test-spec support; `cargo-mutants` as the mutation recipe with its outcome mapping (`caught` → `detection`, `missed` → `survivor`, `unviable` → `invalid`, `timeout` or failing baseline → `inconclusive`); the nested-worktree trap (excluded crates need an empty `[workspace]` table); committing an excluded crate's `Cargo.lock`; a `task verify` example and the Docker-only variant; how to add a Rust specialist contract. `docs/brownfield.md` points to it.
2. Owner rulings 2026-11-22: `ateam mutate` stays Go-only, `cargo-mutants` documented, no wrapper (#385); no shipped Rust contracts (#386).
3. Generated templates (`internal/bootstrap/templates.go`) reword the three Go-only spots language-neutrally with examples (Go: `gofmt`, `go vet`, `revive`; Rust: `rustfmt`, `clippy`; `testing.F` / `proptest`; `ateam mutate` / `cargo-mutants`). ateam's own `AGENTS.md` and rules index keep their Go rows; the repo protocol's mutation step names `cargo-mutants`.
4. Fix stale rows: the adoption table's `ateam test-spec` and `ateam mutate` rows.

## Scope & Non-Goals

- **In-bounds:** `docs/adoption.md`, `docs/brownfield.md`, `.agents/protocol.md`, `internal/bootstrap/templates.go` and its tests, `openspec/specs/harness/spec.md`.
- **Non-goals:** a mutation wrapper or copy-tree runner; language detection in `init`; Rust contracts; the bbstack half of the DoD (one recorded `cargo-mutants` result, hand-edit-free adoption), which lands with the pin bump after v0.4.3.

## Definition of Done

- Fresh `ateam init` output names no Go-only tool without a language label; `ateam doctor --check` passes on it.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
