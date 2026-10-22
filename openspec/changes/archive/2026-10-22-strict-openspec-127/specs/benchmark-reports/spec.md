## MODIFIED Requirements

### Requirement: Deterministic non-overwriting output and read-only checks

Identical validated input bytes/options SHALL produce identical generated files. Output SHALL use anchored operations after destination identity capture and fixed filenames in an exclusively claimed new directory and SHALL not overwrite or delete unrelated data. Write/close failure or cancellation SHALL prevent successful completion. Check mode SHALL perform no writes, compare regenerated facts/provenance with stored generated files, refuse unsafe paths, and leave human interpretation alone.

#### Scenario: Destination parent trust boundary

- **WHEN** the caller provides the destination parent
- **THEN** that parent is trusted and stable during initial creation/open through identity capture, and replacement in the portable mkdir-to-open gap is outside this guarantee
- **AND** after identity capture, output uses anchored operations and detects destination replacement

#### Scenario: Existing or interrupted output

- **WHEN** a destination already exists, becomes unsafe after identity capture, or an owned write fails
- **THEN** export fails without overwriting unrelated content or claiming a complete report
- **AND** any retained incomplete generated set fails check

#### Scenario: Generated drift and human notes

- **WHEN** generated files are missing, tampered, stale or truncated
- **THEN** check fails without repairing them
- **AND** changes to a separate human-authored interpretation file do not cause generated drift or get overwritten

#### Scenario: Private originals unavailable in CI

- **WHEN** a repository reader has only a public summary and evidence hashes
- **THEN** the report does not claim standalone replay or public rederivation
- **AND** the repository check gate uses redistributable fixtures or explicitly available originals rather than fetching private evidence automatically
