## ADDED Requirements

### Requirement: Codex API calibration adapter
The benchmark runner SHALL support API-only Codex execution with fixed Luna model configuration, shared explicit effort, isolated sessions, structured source replies and bounded execution. It SHALL preserve native terminal/accounting evidence, leave unknown values unknown, reject unsupported identity claims and retain unsuccessful attempts during offline replay.

#### Scenario: Same-model Luna A/A
- **WHEN** both arms select identical treatments through the Codex native adapter
- **THEN** they use identical requested model/settings in independent sessions
- **AND** configured selection is not misrepresented as provider-reported model identity

#### Scenario: Accounting and terminal integrity
- **WHEN** native output is malformed, incomplete, contradictory or lacks required accounting or identity evidence
- **THEN** the runner retains the evidence and applicable refusal/stop outcome
- **AND** cached input and reasoning output are never added twice to primary totals

#### Scenario: API-only isolation
- **WHEN** the adapter executes
- **THEN** only the explicitly selected API credential is available to the authorized provider transport
- **AND** subscription fallback, ambient configuration and session reuse are excluded without bypassing host policy
