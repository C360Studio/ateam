## MODIFIED Requirements

### Requirement: Lifecycle hook normalization under 5ms

`ateam hook` SHALL normalize lifecycle events (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) from Claude Code (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Codex CLI (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Gemini CLI (`SessionStart`, `BeforeAgent`, `PreCompress`, `AfterAgent`), and Antigravity (`SessionStart`, `PreInvocation`, `Stop`) over JSON stdio in under 5ms using cached `.ateam/state.json`.

#### Scenario: Warm hook latency thresholds

- **WHEN** doctor evaluates hook latency
- **THEN** `ateam doctor` SHALL measure warm best-of-3 hook latency, emit a warning when latency exceeds `5000 us` (`<5ms` target budget), and fail `--check` only above `25000 us` (`25ms` hard ceiling).

#### Scenario: Mode switch and Ponytail ladder injection

- **WHEN** `ateam hook --event before-prompt` receives a prompt containing `/ponytail ultra` or `/mode ultra`
- **THEN** it updates `.ateam/state.json` to mode `ultra` and returns `additionalContext` (or `injectSteps` when `--agent agy` is set) containing the Ponytail 6-step decision ladder at `ultra` level

#### Scenario: Ordinary prompt without mode switch

- **WHEN** `ateam hook --event before-prompt` receives a prompt with no `/ponytail` or `/mode` command
- **THEN** it returns no `additionalContext` (and an empty `{}` object for `--agent agy`) so the Ponytail ladder is not re-injected on every prompt

#### Scenario: Antigravity SessionStart and Stop hook contract

- **WHEN** `ateam hook` is invoked with `--agent agy` and a JSON payload containing `conversationId`
- **THEN** `session-start` returns `injectSteps` containing a `systemMessage` with the pickup summary and Ponytail ladder, and `turn-end` records `conversationId` in `.ateam/writer.json` and returns `{"decision":"allow"}`

#### Scenario: Automatic compaction handoff

- **WHEN** a pre-compaction event arrives
- **THEN** if `last_handoff` belongs to the same active `session_id` on the same branch, the hook preserves its 5 semantic fields (`goal`, `addresses`, `constraints`, `done_when`, `checks_to_run`) while refreshing `checkpoint_sha`, `branch`, `dirty`, and `auto_compacted: true`
- **AND** if `last_handoff` is nil, stale for a non-SHA reason, or a fallback auto-handoff from a different or unknown session, the hook writes a fallback auto-handoff whose `goal` begins with `UNKNOWN — Auto-saved before context compaction` and returns its pickup context
- **AND** if `last_handoff` is not stale, is not itself a fallback auto-handoff, and belongs to a different or unknown session, the hook leaves it unchanged and returns pickup context naming the session that holds the slot and stating that no auto-save was written

#### Scenario: Session identity and coordination warnings

- **WHEN** session start or MCP pickup detects changed commit identity, a competing writer lease, or an unavailable Hold scan
- **THEN** pickup context discloses each condition and never reports zero holds for an unavailable scan
- **AND** the same session lease (or a lease matching the invoking client process PID or an exited process PID) does not produce a competing-writer warning, and clearing the lease removes that warning
- **AND** once per session (persisted by session ID in `.ateam/mcp-checked-sessions.json` independently of the single writer lease), the hook enumerates `ateam mcp` processes under the same client process tree via `internal/bootstrap` and emits `MCP server PID n (build x) is older than HEAD; restart the client` when a client-tree MCP server has an `unknown` or older build commit than checkout `HEAD`

#### Scenario: Unknown lifecycle event

- **WHEN** an event has no supported lifecycle mapping
- **THEN** the hook returns no context
