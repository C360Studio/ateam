# benchmark-reports Specification

## Purpose
Defines offline experiment report projection, honest accounting, retained replay association, and deterministic non-overwriting export with read-only drift checks.

## Requirements

### Requirement: Offline validated report projection

Report export SHALL derive its factual output from a bounded immutable input snapshot validated by the existing experiment validator and comparison semantics. It SHALL invoke no adapter, grader, native version probe, credential lookup, network request or automatic next experiment. Original evidence SHALL remain unchanged. Summary artifacts SHALL have a distinct kind and version and SHALL NOT be accepted as replayable runs.

#### Scenario: Complete or stopped history

- **WHEN** a valid complete or explicitly stopped experiment is exported
- **THEN** its retained attempts, schedule, pair eligibility, statuses, limits, provenance and denominators are represented without inventing unstarted outcomes
- **AND** a failed experiment can yield a valid report without being described as successful

#### Scenario: Invalid evidence

- **WHEN** input version, digest, schedule, evaluator or native evidence is inconsistent
- **THEN** export fails before creating output and does not weaken the existing validator

### Requirement: Honest accounting and scoped publication

Exported facts SHALL preserve known/unknown usage, accounting basis, cached/reasoning subset semantics and separate execution/grading times. Default publication SHALL exclude raw instructions, source, provider/native traces, diagnostics, credentials, environment, absolute paths and native session IDs. Remaining metadata SHALL be escaped as data, not executable report markup. Costs SHALL remain unknown in v1; reports SHALL not invent ROI, winner, significance, equivalence or activation claims.

#### Scenario: Unknown and zero usage

- **WHEN** one retained attempt has no authoritative normalized usage and another reports zero
- **THEN** the first remains unknown, the second remains zero, and known subtotals and contributing denominators remain explicit
- **AND** unstarted schedule entries do not become unknown retained observations or quality failures

#### Scenario: Private trace and hostile label

- **WHEN** omitted trace fields contain sentinel private data and included labels contain HTML, Markdown or control characters
- **THEN** private fields are absent and labels cannot create report structure, links or apparent findings
- **AND** publication still requires human review; omission is not a universal secret-scrubbing guarantee

### Requirement: Explicit replay association and outcome

Optional retained replay evidence SHALL be validated and bound to original execution identity and inputs before any replay-agreement claim. Missing replay SHALL remain not-provided. Associated changes in recorded grading outcomes SHALL be reported separately without overwriting the original. Incompatible evidence SHALL be refused.

#### Scenario: Matched or changed regrading

- **WHEN** the supplied replay has identical execution identity and sources but refreshed grading metadata
- **THEN** matching outcomes are reported as matched and differing outcomes as different-results with affected attempts
- **AND** neither state asserts cryptographic proof that the grader was executed

#### Scenario: Unrelated replay

- **WHEN** replay changes source, response, schedule, original execution timing, resolved configuration or evaluator/Go-driver identity
- **THEN** export refuses the association before output creation

### Requirement: Deterministic non-overwriting output and read-only checks

Identical validated input bytes/options SHALL produce identical generated files. The caller SHALL provide a trusted destination parent stable during initial creation/open through identity capture; replacement in the portable mkdir-to-open gap is outside this guarantee. After that capture, output SHALL use anchored operations and detect destination replacement. Output SHALL use fixed filenames in an exclusively claimed new directory and SHALL not overwrite or delete unrelated data. Write/close failure or cancellation SHALL prevent successful completion. Check mode SHALL perform no writes, compare regenerated facts/provenance with stored generated files, refuse unsafe paths, and leave human interpretation alone.

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
