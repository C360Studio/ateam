# Design: Test-to-spec attribution

## Proposed contract

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
