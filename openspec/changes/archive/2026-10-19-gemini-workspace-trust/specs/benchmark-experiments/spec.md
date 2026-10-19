## ADDED Requirements

### Requirement: Generated workspace trust
Native Gemini execution SHALL grant session-only trust to its runner-created workspace through the isolated child environment. It SHALL NOT modify the parent environment or global trust records. Managed policy refusal and configured tool restrictions SHALL remain enforced.

#### Scenario: Independent trusted attempts
- **WHEN** a native attempt starts in a fresh generated workspace
- **THEN** its child receives explicit session trust while other attempts and the parent retain separate state
