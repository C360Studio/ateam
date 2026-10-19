## ADDED Requirements

### Requirement: Gemini native structured source
New Gemini benchmark execution SHALL request native JSON schema output containing exactly one string source property. Legacy prose-output configurations SHALL remain replayable with unchanged saved settings but SHALL NOT start new inference. Strict parsing SHALL NOT repair fenced or prose output.

#### Scenario: Native schema is requested
- **WHEN** a new Gemini experiment starts
- **THEN** native generation configuration requests application/json and a schema requiring only source

#### Scenario: Legacy replay remains immutable
- **WHEN** a previously recorded Gemini experiment without schema mode is rescored
- **THEN** its original configuration and native evidence remain unchanged and no model is called

### Requirement: Operational stops precede missing telemetry
Native timeout, cancellation and adapter failures SHALL retain an outcome-derived stop reason rather than being described only as unavailable model identity. Existing artifacts with prior stop reasons SHALL remain replayable when supported by their original evidence.

#### Scenario: Timeout without native output
- **WHEN** a native attempt times out before reporting model identity
- **THEN** the new run records an attempt timeout and starts no further attempt

### Requirement: Reused sessions are rejected before grading
A repeated native session identifier SHALL prevent candidate grading for that attempt.

#### Scenario: Duplicate session contains executable source
- **WHEN** a second native attempt reports an already seen session identifier
- **THEN** its generated source is not compiled or executed and the run stops with session reuse evidence
