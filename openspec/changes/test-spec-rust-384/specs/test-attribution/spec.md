## ADDED Requirements

### Requirement: Rust test declarations

The checker SHALL parse repository `.rs` files outside excluded directories and `target`, independent of Cargo workspace membership, without a Rust toolchain. Functions whose attribute stack holds a supported test attribute SHALL need the same attached citations as Go tests, with attributes and `///` doc comments allowed inside the stack. Unsupported forms SHALL fail closed.

#### Scenario: Inline and integration Rust tests

- **WHEN** `#[test]` and `#[tokio::test]` functions appear in a `#[cfg(test)] mod tests` block and in a `tests/` file, each with and without an attached citation
- **THEN** cited tests pass, uncited and dangling citations fail at their locations, and copies under `target` are ignored

#### Scenario: Rust fixture strings

- **WHEN** a raw string or comment contains `#[test] fn fake()`
- **THEN** no test declaration is created

## MODIFIED Requirements

### Requirement: Deterministic discovery and verification

The checker SHALL discover supported sources offline and return deterministic sorted diagnostics. Verification SHALL propagate attribution failures. Infrastructure failures and cancellation SHALL prevent a successful result.

#### Scenario: Discovery scope and command contract

- **WHEN** the checker is invoked
- **THEN** the following scope and execution rules apply:

The checker SHALL walk the repository offline, including untracked supported test sources, and sort diagnostics by repository-relative path, line, and message. It SHALL exclude dependency/worktree metadata directories `.git`, `.worktrees`, `node_modules`, `vendor`, `.svelte-kit`, `.enjoy-logs`, Cargo `target` at any depth, and repository outputs `ui/build`, `internal/web/dist`, `coverage`, `bin`, `benchmarks/runs`, `.ateam`. A symlink source SHALL be rejected; directory symlinks SHALL NOT be followed. Fixture source strings are not declarations; supported test files under testdata are not exempt. A tree with no supported tests MAY succeed without asserting coverage of unsupported languages.

The CLI SHALL return 0 for valid discovered tests, 1 for attribution/source errors, and 2 for invocation or scan infrastructure failure. Cancellation SHALL stop checking. Go-only and Rust repositories SHALL require no Node runtime or Rust toolchain; Vitest scanning SHALL require Node and the adopting repository's TypeScript installation at `ui/node_modules/typescript`. `task test:spec` SHALL execute the checker and scanner regression fixtures, and `task verify` SHALL invoke `test:spec` without suppressing failures.

#### Scenario: Stable diagnostics

- **WHEN** the same repository tree is checked twice
- **THEN** the same sorted diagnostics are returned

#### Scenario: Verification refuses missing attribution

- **WHEN** a supported unattributed test is added to the tree
- **THEN** test:spec and verify fail
