## ADDED Requirements

### Requirement: Gemini fatal stderr evidence
The runner SHALL retain recognized Gemini turn-limit error evidence without treating absent native usage as zero or executing later attempts. Historical artifacts without this evidence SHALL remain replayable unchanged.

#### Scenario: Stderr-only turn limit
- **WHEN** Gemini exits with code53, empty stdout and an unambiguous bounded native FatalTurnLimitedError envelope on stderr
- **THEN** the runner retains reported error/session details and a local four-turn-limit diagnostic
- **AND** usage stays unknown, the attempt has no grade and no subsequent attempt executes

#### Scenario: Untrusted framing and compatibility
- **WHEN** stderr contains malformed, conflicting or duplicate-key candidate envelopes, streams are truncated, or the process times out or exits with another code
- **THEN** the runner does not synthesize recognized turn-limit evidence
- **AND** replay of newly normalized evidence re-derives it from raw stderr while historical artifacts preserve their original outcomes
