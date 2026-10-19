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

#### Scenario: Automatic compaction handoff

- **WHEN** a pre-compaction event arrives
- **THEN** the hook saves an automatic handoff for the current branch and HEAD (replacing any stale or prior-session handoff rather than re-saving drifted or closed-PR checkpoints) and returns its pickup context

#### Scenario: Session identity and coordination warnings

- **WHEN** session start or MCP pickup detects changed commit identity, a competing writer lease, or an unavailable Hold scan
- **THEN** pickup context discloses each condition and never reports zero holds for an unavailable scan
- **AND** the same session lease (or a lease matching the invoking client process PID or an exited process PID) does not produce a competing-writer warning, and clearing the lease removes that warning

#### Scenario: Unknown lifecycle event

- **WHEN** an event has no supported lifecycle mapping
- **THEN** the hook returns no context

### Requirement: Dual-forge GitHub and Gitea state sync

`ateam` SHALL support both GitHub (`gh` CLI and REST API) and Gitea (`tea` CLI and `/api/v1` REST API) for listing issues, pull requests, and bootstrapping standard workflow labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`). Synchronization SHALL replace the board snapshot with the live remote forge cards (dropping unbacked/demo cards with no forge counterpart while preserving hold reasons and decision summaries on matching card IDs), record `board_synced_at`, reject partial issue/PR failures, and permit a comment-free fallback when optional comment expansion fails. Explicit `pickup` and `board` queries SHALL attempt a bounded remote forge sync when configured (falling back to cache on failure) and surface sync staleness.

#### Scenario: Gitea API sync

- **WHEN** a repository is configured with a Gitea remote or `GITEA_URL`
- **THEN** `ateam` maps Gitea issues and pull requests into unified board cards (`needs-decision`, `backlog`, `in-progress`, `in-review`, `blocked`, `done`)

### Requirement: Durable shared state and five-field handoff

State SHALL round-trip mode and the five handoff fields without inventing board cards, serialize concurrent updates without losing fields, preserve corrupt cache bytes on refresh failure, and render pickup prompts deterministically. PR handoff comments and pickup decision summaries SHALL preserve the recorded handoff and forge decisions.

#### Scenario: Concurrent updates

- **WHEN** independent updates set mode and handoff
- **THEN** both values survive loading the cache

#### Scenario: Stale handoff invalidation on HEAD drift or closed PR

- **WHEN** a recorded handoff's checkpoint SHA no longer matches HEAD, its branch changed, or its referenced PR is closed/merged
- **THEN** the handoff is marked stale, pickup reports the staleness reason instead of injecting the outdated five-field goal as active work, and pre-compact does not re-save the stale handoff

### Requirement: Agent initialization and configuration parity

Initialization SHALL write usable canonical instructions, skill adapters, hook settings, and OpenSpec scaffolding. Parity checks SHALL reject missing AGENTS imports, instruction size overflow, mismatched rule row counts, copied adapter checklists, missing native hook configuration, placeholder workflow content, and documented hook event drift.

#### Scenario: Generated configuration

- **WHEN** a repository is initialized with default templates
- **THEN** its agent configuration passes parity checks

#### Scenario: Build provenance and duplicate server detection in doctor

- **WHEN** `ateam version`, `session-start`, or `ateam doctor` runs
- **THEN** build provenance (`vcs.revision` or `-ldflags` commit) is reported in `ateam version` and the `[ateam]` session-start line, and `ateam doctor` warns when the binary commit is not an ancestor of checkout `HEAD` or when multiple `ateam serve` processes are running concurrently
