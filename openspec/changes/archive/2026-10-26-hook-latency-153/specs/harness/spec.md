## RENAMED Requirements

- FROM: `### Requirement: Lifecycle hook normalization under 5ms`
- TO: `### Requirement: Lifecycle hook normalization under 10ms`

## MODIFIED Requirements

### Requirement: Lifecycle hook normalization under 10ms

`ateam hook` SHALL normalize lifecycle events (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) from Claude Code (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Codex CLI (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Gemini CLI (`SessionStart`, `BeforeAgent`, `PreCompress`, `AfterAgent`), and Antigravity (`SessionStart`, `PreInvocation`, `Stop`) over JSON stdio in under 10ms using cached `.ateam/state.json`.

#### Scenario: Warm hook latency thresholds

- **WHEN** doctor evaluates hook latency
- **THEN** `ateam doctor` SHALL measure warm best-of-3 hook latency, emit a warning when latency exceeds `10000 us` (`<10ms` target budget), and fail `--check` only above `25000 us` (`25ms` hard ceiling).

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

## MODIFIED Requirements

### Requirement: Agent initialization and configuration parity

Initialization SHALL write usable canonical instructions, skill adapters, hook settings, and OpenSpec scaffolding. Parity checks SHALL reject missing AGENTS imports, instruction size overflow, mismatched rule row counts, copied adapter checklists, missing native hook configuration, placeholder workflow content, and documented hook event drift.

#### Scenario: Generated configuration

- **WHEN** a repository is initialized with default templates
- **THEN** its agent configuration passes parity checks

#### Scenario: Shipped review-pr skill

- **WHEN** a repository is initialized with default templates
- **THEN** `.agents/skills/review-pr/SKILL.md` is written with the same body this repository uses, with thin adapters at `.claude/skills/review-pr/SKILL.md` and `.gemini/commands/review-pr.toml`, and parity checks reject a canonical `review-pr` skill that loses its numbered steps or the record grammar markers `reviewed-by:` and `Reviewed at`

#### Scenario: Build provenance and duplicate server detection in doctor

- **WHEN** `ateam version`, `session-start`, or `ateam doctor` runs
- **THEN** build provenance (`vcs.revision` or `-ldflags` commit) is reported in `ateam version` and the `[ateam]` session-start line (or `unknown` when built without VCS metadata), and `ateam doctor` lists running `ateam serve` and `ateam mcp` processes (with their build commits) and warns when the binary has no build provenance (`binary has no build provenance; build with task build`), when the binary commit is behind checkout `HEAD` (`binary <build> is behind HEAD <head>; run task build`) or not an ancestor of checkout `HEAD`, when hook latency exceeds `10000 us` (`<10ms` budget), when a running `ateam mcp` process is stale, or when multiple `ateam serve` processes are running concurrently

#### Scenario: Stale or missing binary build provenance warning at session start

- **WHEN** `ateam hook --event session-start` runs in the ateam checkout (root `go.mod` declares `module github.com/c360studio/ateam`) with a resolved `HEAD` commit
- **THEN** it compares the embedded build commit (`state.BuildRevision()`) against `ReadFastGitHead` without spawning a git subprocess, emits `binary <build> is behind HEAD <head>; run task build` when the binary commit differs from `HEAD`, emits `binary has no build provenance; build with task build` when build provenance is `unknown` or `dev`, and emits neither line when the binary commit matches `HEAD`

#### Scenario: Build provenance checks only the ateam checkout

- **WHEN** the session-start hook or `ateam doctor` runs in a checkout whose root `go.mod` does not declare `module github.com/c360studio/ateam` (or that has no `go.mod`)
- **THEN** neither compares the binary's build commit with the checkout's `HEAD`: the hook prints neither a `binary <build> is behind HEAD` line nor a `binary has no build provenance` line, doctor prints no not-an-ancestor warning and reports `build_provenance: not applicable (binary is not built from this checkout)`, and the module path is read from `go.mod` with a line scan, without spawning a subprocess
- **AND** in the ateam checkout both checks behave as before

#### Scenario: Forced init backs up hand-written harness documents

- **WHEN** `ateam init --force` would replace `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md` and the existing content differs from the template
- **THEN** the existing bytes are first written to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>` beside the file, and the init result lists each backup path under `backed_up`
- **AND** every backup is written, synced and closed before any file is written, so if any backup cannot be written init fails without replacing any file, and a partly written backup is removed
- **AND** a file whose content already equals the template, and any generated file other than those four, gets no backup
- **AND** when init fails after making backups, it still reports their paths, and `backed_up` is an empty list rather than `null` when nothing was backed up
