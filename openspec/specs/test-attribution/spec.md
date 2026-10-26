# test-attribution Specification

## Purpose

Require machine-readable links from supported test declarations to the behavioral requirements they claim to prove. Deterministic resolution checks attribution; semantic coverage remains independent review responsibility.

## Requirements

### Requirement: Attached test citations

Every supported Go or Vitest test declaration SHALL have an immediately preceding comment group containing one or more `// spec: <path>#Requirement: <exact title>` lines. Every citation SHALL be validated. Missing attribution and source parse failures SHALL fail the gate.

#### Scenario: Declaration selection and attachment

- **WHEN** the checker scans source files
- **THEN** the following selection and attachment rules apply:

`ateam test-spec` SHALL parse repository Go `_test.go` files independent of build tags, selecting top-level Test and Fuzz functions using Go's name convention and executable Example functions using Go's output-comment convention. TestMain, helpers, methods, benchmarks, and functions inside source strings SHALL NOT be treated as test declarations. Generated test sources SHALL have the same requirements as handwritten sources. Nested Go subtests SHALL inherit the enclosing test's references.

Vitest declarations in `.test` and `.spec` files with `.ts`, `.tsx`, `.js`, or `.jsx` extensions SHALL be parsed using TypeScript's syntax tree. Direct imports of `it` and `test`, their imported aliases, namespace calls, unbound global `it`/`test`, supported modifiers, and parameterized/conditional declarations SHALL be checked individually. Suite citations SHALL NOT substitute for test citations. Recognized but unsupported dynamic aliases or call forms SHALL fail closed. Arbitrary wrapper modules, other frameworks, `.mjs`/`.cjs`, and runtime expansions of parameterized cases are outside this gate's support and SHALL be documented.

Every selected declaration SHALL have an immediately preceding comment group containing one or more `// spec: <path>#Requirement: <exact title>` lines. A blank line between the comment group and declaration SHALL prevent attachment. Every citation SHALL be validated, including malformed citations next to valid ones. Other prose comments MAY coexist; block comments do not count as citation lines. Source parse failures SHALL produce errors rather than partial acceptance.

#### Scenario: Attribution missing or detached

- **WHEN** a supported test has no attached citation or a blank line separates its citation from its declaration
- **THEN** the gate reports its location and fails

#### Scenario: Aliases and fixture strings

- **WHEN** a Vitest import alias invokes a test and a Go source string contains a fake Test function
- **THEN** the alias invocation requires attribution and the source string does not create a test declaration

### Requirement: Safe requirement resolution

Every citation SHALL resolve exactly one requirement heading in an allowed repository specification. Unsafe paths, malformed citations, missing or ambiguous headings, and unreadable targets SHALL fail validation.

#### Scenario: Reference grammar and boundaries

- **WHEN** the checker resolves a citation
- **THEN** the following target rules apply:

Each citation SHALL resolve exactly one literal `### Requirement: <title>` heading outside fenced code blocks in a UTF-8 file at `openspec/specs/<capability>/spec.md` or `openspec/changes/<change>/specs/<capability>/spec.md`. Archived changes SHALL NOT be accepted. Paths SHALL be normalized repository-relative paths, without absolute paths, traversal, backslashes, or symlink escape. Missing files, missing or duplicate headings, malformed citations, and unreadable targets SHALL fail the gate. Moving an in-flight requirement during archival SHALL require updating citations in the same change.

#### Scenario: Multiple requirements

- **WHEN** a test has two attached citations that each resolve uniquely
- **THEN** both are accepted

#### Scenario: Unresolved or unsafe target

- **WHEN** a citation names a missing or duplicate heading, a heading inside a code fence, or a path escaping the repository
- **THEN** validation fails at the citation location

### Requirement: Deterministic discovery and verification

The checker SHALL discover supported sources offline and return deterministic sorted diagnostics. Verification SHALL propagate attribution failures. Infrastructure failures and cancellation SHALL prevent a successful result.

#### Scenario: Discovery scope and command contract

- **WHEN** the checker is invoked
- **THEN** the following scope and execution rules apply:

The checker SHALL walk the repository offline, including untracked supported test sources, and sort diagnostics by repository-relative path, line, and message. It SHALL exclude dependency/worktree metadata directories `.git`, `.worktrees`, `node_modules`, `vendor`, `.svelte-kit`, `.enjoy-logs`, and repository outputs `ui/build`, `internal/web/dist`, `coverage`, `bin`, `benchmarks/runs`, `.ateam`. A symlink source SHALL be rejected; directory symlinks SHALL NOT be followed. Fixture source strings are not declarations; supported test files under testdata are not exempt. A tree with no supported tests MAY succeed without asserting coverage of unsupported languages.

The CLI SHALL return 0 for valid discovered tests, 1 for attribution/source errors, and 2 for invocation or scan infrastructure failure. Cancellation SHALL stop checking. Go-only repositories SHALL require no Node runtime; Vitest scanning SHALL require Node and the adopting repository's TypeScript installation at `ui/node_modules/typescript`. `task test:spec` SHALL execute the checker and scanner regression fixtures, and `task verify` SHALL invoke `test:spec` without suppressing failures.

#### Scenario: Stable diagnostics

- **WHEN** the same repository tree is checked twice
- **THEN** the same sorted diagnostics are returned

#### Scenario: Verification refuses missing attribution

- **WHEN** a supported unattributed test is added to the tree
- **THEN** test:spec and verify fail

### Requirement: AIP scenario directive

The checker SHALL accept an attached comment whose raw text starts with `//aip:verifies` as a citation carrying one or more space-separated scenario IDs, each resolved against a `#### Scenario: <ID>` header (the ID followed by a space or end of line) outside code fences in a canonical `openspec/specs/*/spec.md` or in-flight `openspec/changes/*/specs/*/spec.md`, never an archived change. The `spec:` citation form SHALL keep its behaviour.

#### Scenario: Known scenario IDs

- **WHEN** a test carries `//aip:verifies DEV-001.1 DEV-001.2` and a canonical or in-flight spec declares both scenario headers
- **THEN** the checker accepts that test's attribution

#### Scenario: Unknown, archived, or malformed directive

- **WHEN** a directive names an ID no canonical or in-flight spec declares (including one declared only under `openspec/changes/archive/`), names no IDs, or is written `// aip:verifies` with a space
- **THEN** the checker reports a diagnostic at the comment line and exits nonzero
