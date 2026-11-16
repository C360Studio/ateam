# test-attribution Specification Delta

## MODIFIED Requirements

### Requirement: Attached test citations

Every supported Go or Vitest test declaration SHALL have an immediately preceding comment group containing one or more `// spec: <path>#Requirement: <exact title>` or `// spec: <capability> / <exact title>` lines. Every citation SHALL be validated. Missing attribution and source parse failures SHALL fail the gate.

#### Scenario: Declaration selection and attachment

- **WHEN** the checker scans source files
- **THEN** the following selection and attachment rules apply:

`ateam test-spec` SHALL parse repository Go `_test.go` files independent of build tags, selecting top-level Test and Fuzz functions using Go's name convention and executable Example functions using Go's output-comment convention. TestMain, helpers, methods, benchmarks, and functions inside source strings SHALL NOT be treated as test declarations. Generated test sources SHALL have the same requirements as handwritten sources. Nested Go subtests SHALL inherit the enclosing test's references.

Vitest declarations in `.test` and `.spec` files with `.ts`, `.tsx`, `.js`, or `.jsx` extensions SHALL be parsed using TypeScript's syntax tree. Direct imports of `it` and `test`, their imported aliases, namespace calls, unbound global `it`/`test`, supported modifiers, and parameterized/conditional declarations SHALL be checked individually. Suite citations SHALL NOT substitute for test citations. Recognized but unsupported dynamic aliases or call forms SHALL fail closed. Arbitrary wrapper modules, other frameworks, `.mjs`/`.cjs`, and runtime expansions of parameterized cases are outside this gate's support and SHALL be documented.

Every selected declaration SHALL have an immediately preceding comment group containing one or more `// spec: <path>#Requirement: <exact title>` or `// spec: <capability> / <exact title>` lines. A blank line between the comment group and declaration SHALL prevent attachment. Every citation SHALL be validated, including malformed citations next to valid ones. Other prose comments MAY coexist; block comments do not count as citation lines. Source parse failures SHALL produce errors rather than partial acceptance.

### Requirement: Safe requirement resolution

Every citation SHALL resolve exactly one requirement heading in an allowed repository specification. Unsafe paths, malformed citations, missing or ambiguous headings, and unreadable targets SHALL fail validation.

#### Scenario: Reference grammar and boundaries

- **WHEN** the checker resolves a citation
- **THEN** the following target rules apply:

Each citation SHALL either be `<path>#Requirement: <title>`, resolving `### Requirement: <title>` in `openspec/specs/<capability>/spec.md` or `openspec/changes/<change>/specs/<capability>/spec.md`, or the shorthand `<capability> / <title>` (where `<capability>` is a single path segment without `/` or `#`), resolving `### Requirement: <title>` in `openspec/specs/<capability>/spec.md` first and, when absent or not found there, across non-archived `openspec/changes/<change>/specs/<capability>/spec.md`. Archived changes SHALL NOT be accepted. Paths SHALL be normalized repository-relative paths, without absolute paths, traversal, backslashes, or symlink escape. Missing files, missing or duplicate headings, malformed citations, and unreadable targets SHALL fail the gate. Moving an in-flight requirement during archival SHALL require updating path citations in the same change.

#### Scenario: Capability shorthand resolution

- **WHEN** a test carries `// spec: <capability> / <title>`
- **THEN** the checker accepts it when `### Requirement: <title>` appears once outside code fences in `openspec/specs/<capability>/spec.md`, or when absent there appears once across non-archived `openspec/changes/*/specs/<capability>/spec.md`, and fails when it resolves zero or multiple times or only under `openspec/changes/archive/`

### Requirement: Diff-scoped attribution

`ateam test-spec --base <ref>` SHALL report a missing or malformed citation only for a supported test whose span (attached comments through its last line) contains a line changed between the merge base of `<ref>` and the working tree, or that lies in an untracked file. Every other diagnostic (including a syntactically valid citation whose target file or requirement heading is missing or duplicate, and source parse errors) SHALL apply to all discovered tests. Without `--base` every discovered test SHALL be checked.

#### Scenario: Untouched malformed citation outside the diff

- **WHEN** an unchanged test at `<ref>` carries a malformed `// spec:` line or spaced `// aip:verifies` line and the working tree changes another test
- **THEN** `ateam test-spec --base <ref>` does not report the unchanged test's malformed citation, while editing that test or running without `--base` reports it
