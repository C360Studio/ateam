# harness Specification

## Purpose

Defines the behavioral contract for `ateam`: universal CLI lifecycle hooks across Claude Code, Codex CLI, and Gemini CLI, the 2-tool low-tax MCP server (`ateam_state`, `ateam_task`), OpenSpec queue and `Hold:` validation (`spec:queue` and `spec:check`), dual-forge state synchronization (**GitHub** and **Gitea**), `// shortcut:` debt scanning, and the agentic benchmark suite (`ateam bench`).

## Requirements

### Requirement: Lifecycle hook normalization under 5ms

`ateam hook` SHALL normalize lifecycle events (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) from Claude Code (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Codex CLI (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), and Gemini CLI (`SessionStart`, `BeforeAgent`, `PreCompress`, `AfterAgent`) over JSON stdio in under 5ms using cached `.ateam/state.json`.

#### Scenario: Mode switch and Ponytail ladder injection

- **WHEN** `ateam hook --event before-prompt` receives a prompt containing `/ponytail ultra` or `/mode ultra`
- **THEN** it updates `.ateam/state.json` to mode `ultra` and returns `additionalContext` containing the Ponytail 6-step decision ladder at `ultra` level

#### Scenario: Ordinary prompt without mode switch

- **WHEN** `ateam hook --event before-prompt` receives a prompt with no `/ponytail` or `/mode` command
- **THEN** it returns no `additionalContext` so the Ponytail ladder is not re-injected on every prompt

#### Scenario: Automatic compaction handoff

- **WHEN** a pre-compaction event arrives
- **THEN** the hook saves an automatic handoff and returns its pickup context

#### Scenario: Session identity and coordination warnings

- **WHEN** session start detects changed commit identity, a competing writer lease, or an unavailable Hold scan
- **THEN** pickup context discloses each condition and never reports zero holds for an unavailable scan
- **AND** the same session lease does not produce a competing-writer warning, and clearing the lease removes that warning

#### Scenario: Unknown lifecycle event

- **WHEN** an event has no supported lifecycle mapping
- **THEN** the hook returns no context

### Requirement: Two-tool MCP context budget

`ateam mcp` SHALL expose exactly two tools (`ateam_state` and `ateam_task`) whose serialized JSON schema is strictly under 300 tokens (~1,200 bytes), alongside zero-turn-tax MCP Prompts and Resources.

#### Scenario: MCP tools/list token check

- **WHEN** a client requests `tools/list` from `ateam mcp`
- **THEN** the response contains `ateam_state` and `ateam_task` and its estimated token count (`bytes / 4`) is `< 300`

### Requirement: OpenSpec queue and stray Hold detection

`ateam state spec-queue` SHALL scan `openspec/changes/*/tasks.md` and report open-task holds (`Hold:`) and caveats (`WONTDO`, `HALT`, `RED`, `BLOCKED`, `OPEN-Q`). `ateam state spec-check` SHALL fail with exit code 1 when any `Hold:` appears below a file's first `## ` heading outside every task block, or exit code 2 when a `tasks.md` is unreadable or invalid UTF-8.

#### Scenario: Stray Hold below heading

- **WHEN** an in-flight change's `tasks.md` has `Hold: waiting on #12` in a paragraph outside any `- [ ]` or `- [x]` task block below `## 1. Tasks`
- **THEN** `ateam state spec-check` reports `<path>:<line>:` and exits non-zero

### Requirement: Dual-forge GitHub and Gitea state sync

`ateam` SHALL support both GitHub (`gh` CLI and REST API) and Gitea (`tea` CLI and `/api/v1` REST API) for listing issues, pull requests, and bootstrapping standard workflow labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`). Synchronization SHALL preserve local/dashboard cards and other forges while replacing the selected forge snapshot, reject partial issue/PR failures, and permit a comment-free fallback when optional comment expansion fails.

#### Scenario: Gitea API sync

- **WHEN** a repository is configured with a Gitea remote or `GITEA_URL`
- **THEN** `ateam` maps Gitea issues and pull requests into unified board cards (`needs-decision`, `backlog`, `in-progress`, `in-review`, `blocked`, `done`)

### Requirement: Constant-time forge webhook verification and rate limiting

`ateam serve` SHALL default to loopback, enforce same-origin and JSON content type on browser mutations, and authenticate GitHub/Gitea webhook requests with constant-time HMAC-SHA256 verification and per-IP fixed-window rate limiting. Authentication SHALL fail closed unless unsigned webhooks are explicitly enabled.

#### Scenario: Webhook defaults and explicit override

- **WHEN** the server starts with default configuration
- **THEN** it binds to `127.0.0.1:8420`, requires `Content-Type: application/json` and same-origin for browser POST mutations, and limits webhook requests to 60 per minute per IP
- **AND** `/api/webhooks/gitea` verifies `X-Gitea-Signature` and `/api/webhooks/github` verifies `X-Hub-Signature-256` using `hmac.Equal` with `ATEAM_WEBHOOK_SECRET`
- **AND** unsigned requests are refused unless `ATEAM_ALLOW_UNSIGNED_WEBHOOKS=1`


#### Scenario: Valid signed Gitea webhook payload

- **WHEN** `POST /api/webhooks/gitea` receives a payload signed with `ATEAM_WEBHOOK_SECRET`
- **THEN** the matching `BoardCard` column and labels update atomically in `.ateam/state.json` under `.ateam/state.lock` and the response reports `merge_allowed` based on fail-closed OpenSpec `Hold:` scanning

### Requirement: Non-destructive Go -overlay mutation checks and PBT discipline

`ateam mutate` (`task mutate:check`) SHALL execute baseline, mutant (`go test -overlay`), after-run, and coverage-reach checks without modifying any file in the repository tree, classifying outcomes strictly as `detection` (exit 0), `survivor` (exit 1), `invalid` (exit 1), or `inconclusive` (exit 1), and refusing invalid inputs (`_test.go`, in-repo mutant, identical mutant, missing `-expect`/`-expect-text`, or `-seed 0`) with exit code 2.

#### Scenario: Mutant detected via Go -overlay without modifying repository tree

- **WHEN** `ateam mutate` runs against a non-test `.go` file and an out-of-tree mutant that causes the named test to fail with the expected assertion
- **THEN** it prints `verdict: detection`, leaves the target file's SHA-256 unchanged, and exits with status 0


### Requirement: Durable shared state and five-field handoff

State SHALL round-trip mode and the five handoff fields without inventing board cards, serialize concurrent updates without losing fields, preserve corrupt cache bytes on refresh failure, and render pickup prompts deterministically. PR handoff comments and pickup decision summaries SHALL preserve the recorded handoff and forge decisions.

#### Scenario: Concurrent updates

- **WHEN** independent updates set mode and handoff
- **THEN** both values survive loading the cache

### Requirement: Fast Git identity across worktrees

Git identity reads SHALL resolve loose and packed refs, detached HEADs, and linked-worktree gitdir/commondir pointers, returning the branch and seven-character commit identity.

#### Scenario: Linked worktree identity

- **WHEN** a linked worktree stores its branch ref in the common git directory
- **THEN** identity reads resolve that branch and commit

### Requirement: Inline shortcut inventory

Shortcut scanning SHALL report the source path, line, limit, and revisit trigger from supported source comment markers.

#### Scenario: Source marker

- **WHEN** a source file contains a shortcut comment with a comma-separated revisit trigger
- **THEN** the inventory preserves its location and both fields

### Requirement: Task target discovery and validation

Task execution SHALL discover top-level targets in the project Taskfile and reject unknown targets before invoking the task runner.

#### Scenario: Unknown target

- **WHEN** a caller requests a target absent from the Taskfile
- **THEN** execution returns an unknown-target error

### Requirement: Agent initialization and configuration parity

Initialization SHALL write usable canonical instructions, skill adapters, hook settings, and OpenSpec scaffolding. Parity checks SHALL reject missing AGENTS imports, instruction size overflow, mismatched rule row counts, copied adapter checklists, missing native hook configuration, placeholder workflow content, and documented hook event drift.

#### Scenario: Generated configuration

- **WHEN** a repository is initialized with default templates
- **THEN** its agent configuration passes parity checks

### Requirement: Board state and authenticated API access

The board SHALL expose state and support card creation, movement, handoff, and mode updates through its API, with the static application served at the root. Without an API token, protected reads and writes SHALL accept only loopback clients; when a token is configured, every client SHALL supply it. Board helpers SHALL group cards into all six canonical columns, filter by text, agent, forge, and column, and format five-field pickup prompts deterministically.

The board SHALL support persisted repository registration, a selected repository, explicit forge settings, canonical OpenSpec requirement details and Current Ops. Mutations SHALL refuse a stale repository selection; failed persistence SHALL not change registration in memory. Explicit forge removal SHALL survive refresh, while automatic detection SHALL follow origin changes. Sync SHALL address the configured target and SHALL not fall back to a different repository after failure. Current Ops SHALL distinguish the selected checkout's inspected dirty state from uninspected worktrees, and compatible checkpoint SHA abbreviations SHALL not imply drift. Dashboard loads SHALL discard superseded results and clear stale state on failure. Dialogs SHALL contain focus, support Escape, and expose submission feedback. Benchmarks SHALL run on demand.

#### Scenario: Repository selection history

- **WHEN** a user adds and selects another repository, then removes it
- **THEN** state and model routing follow that selection and fall back to the startup repository
- **AND** stale clients cannot write to the newly selected repository

#### Scenario: Configured token

- **WHEN** a loopback client requests protected state without the configured token
- **THEN** the request is unauthorized
