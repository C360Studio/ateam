## ADDED Requirements

### Requirement: Versioned Gemini best-effort accounting
New Gemini execution SHALL explicitly select the verified stable CLI0.64.0. Historical0.43.0 configurations and artifacts SHALL remain replayable without changing their settings or comparison interpretation. Reports SHALL distinguish known normalized usage from unknown usage and SHALL NOT claim reconciled cost.

#### Scenario: Explicit stable execution
- **WHEN** a new Gemini experiment is resolved or executed
- **THEN** legacy or unsupported selection is refused before executable probes or credentials
- **AND** the actual CLI and retained version metadata must match the selected stable contract

#### Scenario: Partial retained history
- **WHEN** a stableCLI run contains reported and unknown usage, including failed or unpaired attempts
- **THEN** its report includes the known normalized token subtotal and reported/unknown retained-attempt counts
- **AND** no reported usage yields an unknown subtotal, measuredzero stayszero, cost staysunknown and conservative stop rules remain in force

#### Scenario: Historical replay
- **WHEN** an artifact using omitted or explicit0.43.0 selection is rescored
- **THEN** the historical settings and comparison remain unchanged
- **AND** mismatched retained native-version provenance is refused
