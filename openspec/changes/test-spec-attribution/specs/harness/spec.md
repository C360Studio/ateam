## ADDED Requirements

### Requirement: Deterministic test-to-spec attribution

ATeam SHALL provide an offline deterministic verification gate requiring every supported test declaration to cite one or more specification requirements through attached machine-readable source comments. The gate SHALL cover repository Go tests, fuzz targets, executable examples, and TypeScript Vitest tests, with documented nested-case inheritance and narrowly defined fixture/generated-source handling. Benchmarks and unsupported languages SHALL be explicitly identified as outside the gate's scope.

References SHALL resolve repository-relative specification paths and exact requirement headings. The checker SHALL reject missing or malformed citations, unsafe paths, missing targets, ambiguous headings, and source parse failures with sorted file-and-line diagnostics and nonzero exit status. `task verify` SHALL invoke this gate. Citation validity SHALL NOT be presented as proof of semantic test coverage.

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
