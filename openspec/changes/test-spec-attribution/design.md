# Design: Test-to-spec attribution

## Contract

Use one or more source comments of the form:

```go
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestOpenSpecQueueAndCheck(t *testing.T) { /* ... */ }
```

The fragment is the exact `### Requirement:` heading text, not a rendered Markdown slug. Multiple comment lines represent multiple requirements. Resolve repository-relative paths offline and reject traversal, absolute paths, symlink escape, missing files, and missing or duplicate target headings. Canonical specs and in-flight delta specs are allowed; archive moves must update references in the same change.

The implementation must first inventory declaration shapes in Go and Vitest and finalize attachment rules with fixtures. Top-level Go Test/Fuzz declarations and executable examples require attribution. Named nested cases may inherit their enclosing test's citations; a Vitest describe block alone must not satisfy unrelated tests. Benchmarks are outside the behavioral-test gate and must be documented as such. Fixture strings and generated sources need explicit, narrow handling rather than a broad directory bypass. Unsupported languages must be reported honestly in adoption guidance.

## Implementation approach

Prefer Go standard-library parsing for Go declarations and existing installed parsing facilities for TypeScript. Do not recognize declarations with a permissive text search that accepts comments or fixture strings as real tests. Keep diagnostics sorted by repository-relative file, line, and message; the same tree produces identical output. Require a nonzero exit on any attribution or parse error. Add a Taskfile gate to `verify`, with explicit usage for adopting projects.

Backfill real requirement citations. Add missing behavioral requirements where tests currently have no appropriate contract; avoid attaching every test to a broad unrelated requirement merely to pass the gate. Review generated bootstrap instructions alongside repository rules.

## Verification design

Use fixtures with hand-authored expected diagnostics for missing comments, multiple references, malformed syntax, ambiguous headings, nested cases, fixture text, and escaping paths. Establish failing regressions before checker implementation. Because attachment and reference parsing interact, use generated inputs where they exercise grammar boundaries; otherwise record why a finite fixture matrix covers the relevant combinations. Prove the missing-attribution and unresolved-reference refusals with non-destructive mutations. Validate full verification wiring and parity before marking implementation complete.

## Final discovery and execution boundaries

Canonical behavior is specified in `openspec/specs/test-attribution/spec.md`. Node and the existing TypeScript dependency are used only when JS/TS test files exist; the scanner is embedded in the Go binary and resolves TypeScript at the target repository's `ui/node_modules/typescript`. No dependency was added. Scanner regression tests are Vitest `.test.js` files, so they are themselves subject to attribution and run explicitly in `task test:spec`.

Go AST parsing includes all build tags and generated `_test.go` files. Source strings, TestMain, benchmarks, helpers, and non-executable examples are excluded. Vitest aliases and direct namespace/global calls are supported; dynamic escape of recognized test bindings fails closed. Arbitrary wrapper modules and runtime expansion counts are not claimed. Directory symlinks are not traversed; symlink test files fail. Exact discovery exclusions, attachment behavior, and exit codes are in the canonical spec and protocol.

The independent plan review found no architectural blocker and requested explicit discovery, unsafe-path, comment-gap, fixture, alias, and verification-wiring tests. These are acceptance cases, not optional follow-up work.
