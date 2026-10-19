## ADDED Requirements

### Requirement: Claude API calibration adapter
The benchmark runner SHALL support Claude Code API-key authentication with a fixed Haiku model, shared explicit effort across arms, isolated sessions, bounded execution, inspectable configuration and no subscription fallback. It SHALL preserve native per-model usage and terminal evidence, normalize cache tokens exactly once and leave missing accounting unknown. It SHALL reject unexpected models and preserve failed attempts during offline replay.

#### Scenario: Same-model Haiku A/A
- **WHEN** identical treatments select the Claude native adapter
- **THEN** both arms use the same configured model and effort in independent sessions
- **AND** only explicit API credentials reach the isolated child

#### Scenario: Unusable accounting
- **WHEN** Claude fails or returns incomplete, inconsistent or unexpected-model accounting
- **THEN** the runner retains the terminal evidence and applicable stop reason
- **AND** it makes no unsupported quality or billing claim

#### Scenario: Policy and authentication preflight
- **WHEN** Claude runs on a managed installation or without the required API credential
- **THEN** managed policy is never bypassed and missing credentials never fall back to a subscription
- **AND** the live calibration requires recorded policy preflight

#### Scenario: Structured replies and legacy replay
- **WHEN** a new Claude experiment executes
- **THEN** its explicit json-schema contract requires a nonempty source in native structured_output without prose fallback
- **AND** legacy text-output artifacts remain replayable with unchanged failure outcomes but cannot launch new model calls
