# Proposal: test-spec-rust-384
Approved-by: Coby Leuschke (2026-11-22) sha256:7bec170ee188

## Track

`design`: adds a spec Requirement (`test-attribution`) and modifies one in `harness` (two capabilities); `ateam test-spec` behaviour is reachable from adopting repositories.

## Why

`ateam test-spec` selects Go `_test.go` and Vitest `.test`/`.spec` files only (`internal/testspec/check.go:94-104`), so a Rust adopter (#382, test bed bbstack) cannot gate test-to-spec attribution at all: every `#[test]` is invisible and the gate passes vacuously.

## What Changes

1. Discover `.rs` files anywhere outside excluded directories (adding `target`), independent of Cargo workspace membership, and select functions whose attribute stack carries an attribute whose path ends in `test` (`#[test]`, `#[tokio::test]`, `#[async_std::test]`, any arguments), in inline `#[cfg(test)] mod` blocks and `tests/*.rs` alike.
2. Attach citations with the Go rule: the contiguous `//` line-comment group directly above the declaration; attributes and `///` doc comments inside the stack do not break attachment, a blank line does.
3. Parse with a small Go-native Rust lexer (comments, nested block comments, strings, raw strings, char literals, braces) so tests inside strings or macro bodies are handled deterministically, no Rust toolchain required. `--base` scoping uses the function's closing brace.
4. New Requirement *Rust test declarations* in `test-attribution`; *Deterministic test-to-spec attribution* in `harness` names Rust; protocol § *Test-to-spec attribution* and `docs/adoption.md` name Rust and its limits.

## Scope & Non-Goals

- **In-bounds:** `internal/testspec/` (new `rust.go`, `check.go` walk), fixtures in `internal/testspec/check_test.go`, the two specs, `.agents/protocol.md`, `internal/bootstrap/templates.go` protocol copy, `docs/adoption.md`.
- **Non-goals:** pytest; macro-generated or `rstest`/`test_case` expansions beyond the ruled attribute set; running `cargo`; shell/curl integration tests; the bbstack CI wiring (lands with the pin bump after release).

## Definition of Done

- Fixtures per #384: attributed `#[test]`, attributed `#[tokio::test]`, unattributed (fails), dangling heading (fails), each inline `mod tests` and in `tests/`; plus a crate outside the workspace, a `target/` copy ignored, and a fake test in a raw string ignored.
- Lexer fuzz target (no panic, deterministic declarations); `task mutate:check` detection on attachment and selection.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
