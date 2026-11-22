## MODIFIED Requirements

### Requirement: Accurate lab positioning and adoption boundaries

The README and adoption documentation SHALL identify ATeam as an opinionated lab primarily grounded in Go development. They SHALL distinguish language-independent coordination mechanisms from Go-specific tooling, project-owned verification tasks, and unvalidated language support. Relevant generated instructions SHALL reflect the same limitations.

#### Scenario: Adopting in a non-Go repository

- **WHEN** a reader evaluates ATeam for a non-Go project
- **THEN** the documentation explains that initialization does not detect or adapt to the project language, mutation tooling requires non-test Go source, shortcut scanning uses an explicit extension allowlist, and the project must supply its own Taskfile and verification commands

#### Scenario: Adopting in a Rust repository

- **WHEN** a reader adopts ATeam in a Cargo repository
- **THEN** the adoption guide names what works unchanged, the Rust debt-scan and test-attribution support, `cargo-mutants` as the mutation recipe with its outcome mapping, the excluded-crate worktree fix, and a `task verify` example, and generated instructions label Go-only tools as examples

#### Scenario: Interpreting benchmark and discipline claims

- **WHEN** a reader evaluates Ponytail modes and benchmark results
- **THEN** the documentation distinguishes injected agent guidance from enforced behavior and describes reference-fixture self-tests without presenting them as proof of improved agent performance
