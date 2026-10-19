## MODIFIED Requirements

### Requirement: Lifecycle hook normalization under 5ms

`ateam hook` SHALL normalize lifecycle events (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) from Claude Code (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Codex CLI (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Gemini CLI (`SessionStart`, `BeforeAgent`, `PreCompress`, `AfterAgent`), and Antigravity (`SessionStart`, `PreInvocation`, `Stop`) over JSON stdio in under 5ms using cached `.ateam/state.json`.

#### Scenario: Mode switch and Ponytail ladder injection

- **WHEN** `ateam hook --event before-prompt` receives a prompt containing `/ponytail ultra` or `/mode ultra`
- **THEN** it updates `.ateam/state.json` to mode `ultra` and returns `additionalContext` (or `injectSteps` when `--agent agy` is set) containing the Ponytail 6-step decision ladder at `ultra` level

#### Scenario: Ordinary prompt without mode switch

- **WHEN** `ateam hook --event before-prompt` receives a prompt with no `/ponytail` or `/mode` command
- **THEN** it returns no `additionalContext` (and an empty `{}` object for `--agent agy`) so the Ponytail ladder is not re-injected on every prompt

#### Scenario: Antigravity SessionStart and Stop hook contract

- **WHEN** `ateam hook` is invoked with `--agent agy` and a JSON payload containing `conversationId`
- **THEN** `session-start` returns `injectSteps` containing a `systemMessage` with the pickup summary and Ponytail ladder, and `turn-end` records `conversationId` in `.ateam/writer.json` and returns `{"decision":"allow"}`
