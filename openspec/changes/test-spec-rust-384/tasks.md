# Tasks: test-spec-rust-384

## 1. Approval

- [ ] 1.1 Owner approves proposal.md and design.md together. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Add `target` to the walk's excluded names and route `.rs` files to `scanRust` (premise: `internal/testspec/check.go:85-110`). Check: a `target/` fixture copy yields no declaration.
- [ ] 2.2 Add `internal/testspec/rust.go`: lexer + selection per design D1–D3, emitting `declaration{File, Line, End, Name, Comments}` (premise: `check.go:37-44`, `scanGo` at `check.go:211`). Check: `#[test]`, `#[tokio::test(flavor = "multi_thread")]`, `#[ignore]`/`#[should_panic]` stacks, nested `mod`, raw-string fake test.
- [ ] 2.3 Unsupported forms fail closed per D4 (premise: `declaration.Error` handling at `check.go:141-144`). Check: `#[test]` before a non-`fn` item and a `$name` macro test report errors.
- [ ] 2.4 Fixture tests in `internal/testspec/check_test.go` per the proposal's DoD, including `--base` scoping by closing brace (premise: `diff.go:20-33`); `FuzzScanRust` over the lexer.
- [ ] 2.5 Mutation checks: drop the attribute-stack skip and the raw-string state; both detect.

## 3. Docs and specs

- [ ] 3.1 Update `.agents/protocol.md:67-80` and its template copy in `internal/bootstrap/templates.go`, and `docs/adoption.md:60`. Check: fresh `ateam init` fixture shows Rust.
- [ ] 3.2 `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
