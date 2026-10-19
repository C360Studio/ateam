## ADDED Requirements

### Requirement: Explicit Gemini output allowance
Gemini experiments MAY set shared `gemini.max_output_tokens` to an integer from 1 through 65536. Omission or null SHALL preserve the historical 8192 allowance and absent-field serialization. Both generation override scopes SHALL request the same effective allowance for both arms. Dry runs and retained settings SHALL expose that allowance; invalid values SHALL fail before native CLI inspection or inference.

#### Scenario: Explicit allowance is delivered
- **WHEN** a Gemini experiment selects 32768 output tokens
- **THEN** both core and selected-model generation overrides request 32768 while model, reasoning and other controls remain unchanged

#### Scenario: Invalid allowance is rejected
- **WHEN** an experiment specifies zero, a negative value, more than 65536 or a non-integer allowance
- **THEN** configuration resolution refuses it before inspecting the native executable or reading credentials

#### Scenario: Historical evidence remains replayable
- **WHEN** an older experiment omits the output allowance
- **THEN** regenerated native settings retain 8192 and serialization omits the new field, preserving historical artifact validation and offline replay

#### Scenario: Retained allowance disagrees with configuration
- **WHEN** saved native generation settings or original configuration have been altered to a different output allowance
- **THEN** execution and retained-evidence validation refuse the inconsistent provenance rather than silently accepting the new allowance

