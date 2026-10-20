## ADDED Requirements

### Requirement: Default client context compaction limits

ATeam initialization SHALL scaffold the context and compaction defaults specified below into newly created client configurations while preserving pre-existing client files unchanged. Parity checks SHALL validate configured fields, reject out-of-range or non-numeric values, and permit valid custom or omitted fields in preserved configurations.

#### Scenario: Newly initialized repository context limits

- **WHEN** `ateam init` creates `.claude/settings.json`, `.codex/config.toml`, and `.gemini/settings.json` in a repository without pre-existing client configs
- **THEN** `.claude/settings.json` sets `autoCompactWindow` to `500000`, `.codex/config.toml` sets top-level `model_context_window = 500000` and `model_auto_compact_token_limit = 450000`, and `.gemini/settings.json` sets `model.chatCompression.contextPercentageThreshold` to `0.5`

#### Scenario: Invalid configured context limit rejected by parity check

- **WHEN** a project client configuration specifies a non-positive or out-of-range context limit (such as `autoCompactWindow < 100000` or `> 1000000`, `model_auto_compact_token_limit > model_context_window`, or `contextPercentageThreshold` outside `(0, 1]`)
- **THEN** `CheckParity` reports a violation naming the file and field
