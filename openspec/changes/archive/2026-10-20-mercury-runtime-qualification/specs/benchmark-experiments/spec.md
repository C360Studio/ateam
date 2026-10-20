## ADDED Requirements

### Requirement: Mercury runtime qualification boundary

Mercury experiments SHALL use a separately qualified native-runtime contract that records CLI, bridge, provider, model and effective controls, preserves unknown evidence and refuses unsupported settings.

#### Scenario: Unqualified existing native adapter

- **WHEN** Mercury is requested through the current fixed Claude or Codex contract
- **THEN** execution is refused rather than silently changing provider, model or wire protocol
- **AND** a direct API success is not treated as native-runtime qualification

#### Scenario: Control translation loses information

- **WHEN** a bridge drops, changes or cannot prove a requested effort, output allowance or tool control
- **THEN** the configuration is not accepted as a fixed-control experiment
- **AND** translated zero usage does not establish a provider-reported zero when the source field was missing

#### Scenario: Offline qualification precedes spending

- **WHEN** a new Mercury runtime path is investigated
- **THEN** isolated offline boundary, failure, accounting and replay checks plus independent review precede a separately authorized live canary and A/A
- **AND** existing Gemini evidence and budgets do not authorize new execution
