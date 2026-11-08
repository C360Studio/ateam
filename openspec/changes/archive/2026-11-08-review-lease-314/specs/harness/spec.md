## ADDED Requirements

### Requirement: Read-only review session writes no writer lease
When `ATEAM_ROLE=review` is set in its environment, `ateam hook` SHALL NOT write or refresh the writer lease on `session-start`, `before-prompt`, or `turn-end`, and SHALL otherwise behave as for any session. Any other value, or no value, SHALL leave lease behavior unchanged.

#### Scenario: Reviewer in a writer's checkout
- **WHEN** a checkout holds another session's writer lease and a session with `ATEAM_ROLE=review` runs session-start, before-prompt, and turn-end there
- **THEN** the lease file is byte-for-byte unchanged

#### Scenario: Role unset or another value
- **WHEN** `ATEAM_ROLE` is unset or is any value other than `review`
- **THEN** the three events write the lease as before
