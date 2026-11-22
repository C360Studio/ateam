# Design: test-spec-rust-384

Drafted by claude (claude-opus-5-5, high) as architect; read-only on code. Owner rules on the marked questions.

## Context

- `internal/testspec/check.go:54-110`: `Check` walks the tree, skips names `.git .worktrees node_modules vendor .svelte-kit .enjoy-logs` (`:85`) and fixed paths (`:88`), and selects only `_test.go` (`:94`) and `.test|.spec.{ts,tsx,js,jsx}` (`:96-100`). Symlinked sources become a `declaration.Error` (`:105-108`).
- `check.go:37-44`: `declaration{File, Line, End, Name, Comments, Error}` is the language-neutral seam; citation and `//aip:verifies` checking (`:136-197`) only reads `Comments[].Text` and needs `//`-prefixed lines.
- `check.go:211-250`: `scanGo` uses `go/parser`; attachment is `fn.Doc` ending on the line before `func` (`:242`), so a blank line detaches.
- `check.go:117-135`: Vitest runs `vitest.mjs` under `node` with the adopter's TypeScript: a precedent for an external parser, and for the error that follows when the runtime is absent (exit 2).
- `diff.go:20-33`: `--base` scoping touches lines `Comments[0].Line .. max(End, Line)`; a Rust scanner must report `End` (closing brace) for this to work.
- `openspec/specs/test-attribution/spec.md:11,22`: attachment rule is "immediately preceding comment group"; block comments never count. `:69`: excluded-directory list has no `target`.
- `.agents/protocol.md:67-80` and its emitted copy `internal/bootstrap/templates.go:303-305` name Go and Vitest only; `docs/adoption.md:60` likewise.
- `go.mod` carries no Rust or tree-sitter dependency today. Sibling precedent: semsource parses Rust with `github.com/smacker/go-tree-sitter` (its `go.mod:16`), built with `CGO_ENABLED=1` (its `Dockerfile:20`), which is option C below.

## Goals / Non-Goals

Goals: every `#[test]`/`#[tokio::test]` function in a Rust tree needs an attached citation, with the Go attachment rule, `--base` scoping, deterministic output, and no new runtime or module dependency.

Non-goals: semantic Rust parsing, macro expansion, pytest, invoking `cargo`, bbstack CI wiring.

## Decisions

### D1. Parser strategy

| Option | Cost | Blast radius | Forecloses |
| --- | --- | --- | --- |
| **A. Go-native lexer** (~200 lines): tokenises comments (`//`, `///`, nested `/* */`), strings, raw strings `r#".."#`, byte/char literals vs lifetimes, `#[...]` attributes, `fn`, braces | One file + fuzz target | `internal/testspec` only | Exact handling of macro-generated tests (detected only as unsupported, D4) |
| B. `syn`-based helper run via `cargo` (like `vitest.mjs`) | Rust source to vendor, build per run, toolchain required | Adopters need `cargo`; CI image grows; exit 2 when missing | Zero-dependency gate; offline `ateam` binary story |
| C. tree-sitter-rust Go binding | New cgo module | Release builds need cgo on every platform | Static single-binary release |

Adopter seam: with A, a Rust repo needs nothing installed beyond the `ateam` binary; with B, `cargo` becomes a gate prerequisite. Doing nothing leaves Rust tests silently unchecked. **Recommended: A** (issue #384 notes no toolchain dependency today).

### D2. Which attributes select a test

| Option | Effect | Risk |
| --- | --- | --- |
| **A. Exact set**: `test`, `tokio::test` (any arguments) | Matches #384's minimum | `#[async_std::test]`, `#[rstest]`, `#[test_case(..)]` go unchecked silently |
| B. Any path whose last segment is `test` | Picks up `async_std::test`, `actix_rt::test`, `sqlx::test` | A non-test proc-macro named `test` would need a citation |
| C. A, plus fail closed on known test-generating macros (`rstest`, `test_case`, `wasm_bindgen_test`) | No silent gaps for the common crates | Adopters using them cannot pass the gate until supported |

**Ruled B (owner, 2026-11-22):** any attribute path whose last segment is `test` (`test`, `tokio::test`, `async_std::test`, `sqlx::test`, with any arguments), documented as such.

### D3. Attachment

Rust stacks attributes and doc comments between prose and `fn`. Options: (A) the contiguous run of `//` line comments, `///` doc comments and `#[...]` attributes ending at `fn` (no blank line anywhere in the run) is the group, and `// spec:` lines anywhere in it count; (B) only `//` lines directly above the *first* attribute count. A accepts `#[test]\n// spec: ..\nfn x()` as well as the usual form; B mirrors Go literally but rejects a citation placed between attributes, which formatters do not move. **Recommended: A.** `///` and `//!` lines never count as citations (they trim to `/ spec:`), matching the block-comment rule. Visibility and qualifiers (`pub`, `async`, `unsafe`, `const`, `extern "C"`) may sit between the last attribute and `fn`.

### D4. Fail-closed forms

A test attribute followed by something other than an `fn` item, a function named by a macro metavariable (`fn $name`), or an unterminated string/comment/brace produces a `declaration.Error` (exit 1), the same as a Go parse error (`check.go:214-220`). Alternative: skip silently. Skipping keeps `macro_rules!` test generators passing but hides them; failing closed matches the existing Vitest rule for unsupported call forms. **Ruled fail closed (owner, 2026-11-22)**, with the protocol naming the escape hatch (cite a hand-written wrapper test).

### D5. Declaration name and scope

`Name` is the module path inside the file (`tests::parses_frame`) so diagnostics stay unique per file; `End` is the line of the matching `}`. Every `.rs` file is scanned regardless of `src/`, `tests/`, `benches/`, `examples/` or workspace membership, so the bbstack crate excluded from its workspace is covered by path, not by `Cargo.toml`. `target` joins the excluded names at any depth (spec `:69` list).

## Risks / Trade-offs

- A lexer can mis-tokenise rare syntax (raw identifiers `r#fn`, lifetimes vs char literals `'a'`/`'a`, shebang lines). Mitigation: fuzz target asserting no panic and stable output, plus fixtures for each; a mis-tokenised file fails closed (D4) rather than passing.
- `#[cfg_attr(test, test)]` and similar indirect forms are not recognised; documented as unsupported.
- Repos with many uncited Rust tests turn red on adoption; `--base` covers brownfield (`diff.go`), as for Go.

## Migration Plan

No data migration. Released in v0.4.3; adopters on a pin bump see Rust tests become gated. bbstack adds `ateam test-spec --base origin/main` to its verify after the bump. Rollback is reverting the walk branch in `check.go`.

## Open Questions

1. Resolved: D2 B and D4 fail closed (owner, 2026-11-22).
2. Non-blocking: should `benches/` `#[bench]` functions be selected? Draft: no, matching Go's benchmark exclusion.
