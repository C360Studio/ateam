## ADDED Requirements

### Requirement: AIP scenario directive

The checker SHALL accept an attached comment whose raw text starts with `//aip:verifies` as a citation carrying one or more space-separated scenario IDs, each resolved against a `#### Scenario: <ID> ` header outside code fences in a canonical `openspec/specs/*/spec.md` or in-flight `openspec/changes/*/specs/*/spec.md`, never an archived change. The `spec:` citation form SHALL keep its behaviour.

#### Scenario: Known scenario IDs

- **WHEN** a test carries `//aip:verifies DEV-001.1 DEV-001.2` and a canonical or in-flight spec declares both scenario headers
- **THEN** the checker accepts that test's attribution

#### Scenario: Unknown, archived, or malformed directive

- **WHEN** a directive names an ID no canonical or in-flight spec declares (including one declared only under `openspec/changes/archive/`), names no IDs, or is written `// aip:verifies` with a space
- **THEN** the checker reports a diagnostic at the comment line and exits nonzero
