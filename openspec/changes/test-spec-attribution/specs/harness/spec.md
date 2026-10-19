## ADDED Requirements

### Requirement: Deterministic test-to-spec attribution

ATeam SHALL enforce attached machine-readable requirement citations for supported Go and Vitest tests through an offline deterministic gate in `task verify`. Missing, malformed, unsafe, or unresolved citations and parse failures SHALL fail verification. Attribution SHALL NOT be presented as semantic coverage proof. Detailed discovery, syntax, and resolution rules are specified by the test-attribution capability.

#### Scenario: Valid multiple requirement citations

- **WHEN** a supported test has attached valid citations to two unique requirements within the repository
- **THEN** the checker accepts that test's attribution and produces deterministic results

#### Scenario: Missing or unresolved attribution

- **WHEN** a supported test has no citation or cites a missing or ambiguous requirement
- **THEN** the checker reports the test or citation location and exits nonzero

#### Scenario: Unsafe reference or malformed source

- **WHEN** a citation escapes the repository or a supported source file cannot be parsed
- **THEN** the checker refuses the input without accepting incomplete scan results

#### Scenario: Verification integration

- **WHEN** task verify runs on a tree with an unattributed supported test
- **THEN** verification fails on the attribution gate
