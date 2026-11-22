## ADDED Requirements

### Requirement: Rust test declarations

The checker SHALL parse repository `.rs` files outside excluded directories and `target`, independent of Cargo workspace membership, without a Rust toolchain. Functions whose attribute stack holds a supported test attribute SHALL need the same attached citations as Go tests, with attributes and `///` doc comments allowed inside the stack. Unsupported forms SHALL fail closed.

#### Scenario: Inline and integration Rust tests

- **WHEN** `#[test]` and `#[tokio::test]` functions appear in a `#[cfg(test)] mod tests` block and in a `tests/` file, each with and without an attached citation
- **THEN** cited tests pass, uncited and dangling citations fail at their locations, and copies under `target` are ignored

#### Scenario: Rust fixture strings

- **WHEN** a raw string or comment contains `#[test] fn fake()`
- **THEN** no test declaration is created
