# harness Specification

## Purpose

Defines the behavioral contract for `ateam`: universal CLI lifecycle hooks across Claude Code, Codex CLI, and Gemini CLI, the 2-tool low-tax MCP server (`ateam_state`, `ateam_task`), OpenSpec queue and `Hold:` validation (`spec:queue` and `spec:check`), dual-forge state synchronization (**GitHub** and **Gitea**), `// shortcut:` debt scanning, and the agentic benchmark suite (`ateam bench`).
## Requirements
### Requirement: Lifecycle hook normalization under 10ms

`ateam hook` SHALL normalize lifecycle events (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) from Claude Code (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Codex CLI (`SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`), Gemini CLI (`SessionStart`, `BeforeAgent`, `PreCompress`, `AfterAgent`), and Antigravity (`SessionStart`, `PreInvocation`, `Stop`) over JSON stdio in under 10ms using cached `.ateam/state.json`.

#### Scenario: Warm hook latency thresholds

- **WHEN** doctor evaluates hook latency
- **THEN** `ateam doctor` SHALL measure warm best-of-3 hook latency, emit a warning when latency exceeds `10000 us` (`<10ms` target budget), and fail `--check` only above `100000 us` (`100ms` hard ceiling).

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

### Requirement: Two-tool MCP context budget

`ateam mcp` SHALL expose exactly two tools (`ateam_state` and `ateam_task`) whose serialized JSON schema is strictly under 300 tokens (~1,200 bytes), alongside zero-turn-tax MCP Prompts and Resources.

#### Scenario: MCP tools/list token check

- **WHEN** a client requests `tools/list` from `ateam mcp`
- **THEN** the response contains `ateam_state` and `ateam_task` and its estimated token count (`bytes / 4`) is `< 300`

#### Scenario: MCP server build provenance and stale binary warning in pickup

- **WHEN** `ateam mcp` handles a `pickup` tool call or resource read
- **THEN** it captures `state.BuildRevision()` at server startup, reports `build=<disk-sha> mcp=<server-sha>` in the `[ateam]` pickup header line (`build=absent` when there is no `./bin/ateam`, `build=unreadable` when its `--version` fails or names no build, in which case staleness is judged against checkout `HEAD`; `ateam doctor` reports the same label as `disk_build` and warns on `unreadable`), and appends `WARNING: MCP server binary is stale; restart the client to pick up <sha>` when the running server's build commit differs from the `./bin/ateam` binary on disk or from checkout `HEAD`

#### Scenario: CLI and MCP report the release version from the build

- **WHEN** `ateam version` runs or `ateam mcp` answers `initialize`
- **THEN** both report the same release version, which is the `-ldflags` value of `state.Version` when one is stamped, else the main module version embedded by the Go toolchain when it is neither empty nor `(devel)`, else `dev`, so a build of a release tag reports that tag without a source edit; `task build` stamps `state.Version` from `git describe --tags --always --dirty`, so a build in a linked worktree reports its own checkout rather than the parent checkout Go's VCS stamping would read
- **AND** MCP `serverInfo.version` reads `<version> (<build>)`, with the same build label as `ateam version`

### Requirement: OpenSpec queue and stray Hold detection

`ateam state spec-queue` SHALL scan `openspec/changes/*/tasks.md` and report open-task holds (`Hold:`) and caveats (`WONTDO`, `HALT`, `RED`, `BLOCKED`, `OPEN-Q`). `ateam state spec-check` SHALL fail with exit code 1 when any `Hold:` appears below a file's first `## ` heading outside every task block, or exit code 2 when a `tasks.md` is unreadable or invalid UTF-8. Truncated caveat display text SHALL remain valid UTF-8.

#### Scenario: Stray Hold below heading

- **WHEN** an in-flight change's `tasks.md` has `Hold: waiting on #12` in a paragraph outside any `- [ ]` or `- [x]` task block below `## 1. Tasks`
- **THEN** `ateam state spec-check` reports `<path>:<line>:` and exits non-zero

#### Scenario: Multibyte text at the display cut

- **WHEN** a caveat's or hold's text crosses the 104-byte display limit
- **THEN** the displayed text ends before any split multibyte character or partial word (truncating on a word boundary with `…`), is valid UTF-8, and renders a single `Hold:` prefix derived from the captured hold text without re-prefixing the task's leading word or number

#### Scenario: Completed unarchived OpenSpec change

- **WHEN** a change directory remains under `openspec/changes/<id>` with all tasks completed (`N/N`, `N > 0`) and zero caveats
- **THEN** `ateam state spec-queue` prints `done, not archived — move openspec/changes/<id> to openspec/changes/archive/<date>-<id>` instead of `ok`, `BuildPickupSummary` lists it separately under `OpenSpec completed changes (done, not archived)` rather than `OpenSpec in-flight changes`, and `ateam doctor` emits a warning naming the unarchived change

#### Scenario: Proposal awaiting owner approval

- **WHEN** a change's `proposal.md` has an empty `Approved-by:` line and its first open task carries `Hold: OPEN-Q proposal awaits owner approval`, every other task being done
- **THEN** `ateam state spec-queue` reports the change with an empty `approved_by` and flags the hold rather than printing `done, not archived`
- **AND** once the owner fills `Approved-by: <name> (<YYYY-MM-DD>)` and removes the hold, `approved_by` carries that value; an `Approved-by:` line inside a code fence is not the header

#### Scenario: Proposal over one page

- **WHEN** a change's `proposal.md` has 61 or more lines
- **THEN** `ateam state spec-check` prints `warning: openspec/changes/<id>/proposal.md has <n> lines, over the one-page limit of 60` and still exits 0; at 60 lines it prints no warning

### Requirement: Dual-forge GitHub and Gitea state sync

`ateam` SHALL support both GitHub (`gh` CLI and REST API) and Gitea (`tea` CLI and `/api/v1` REST API) for listing issues, pull requests, and bootstrapping standard workflow labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`).

#### Scenario: Gitea API sync

- **WHEN** a repository is configured with a Gitea remote or `GITEA_URL`
- **THEN** `ateam` maps Gitea issues and pull requests into unified board cards (`needs-decision`, `backlog`, `in-progress`, `in-review`, `blocked`, `done`)

#### Scenario: Reconciled bounded forge queries

- **WHEN** a configured forge is synchronized or pickup/board queries request its state
- **THEN** Synchronization SHALL reconcile the synced provider's cards into the board (dropping legacy demo cards and stale remote cards for that provider while preserving legitimate local dashboard cards, cards from the other forge, and hold reasons and decision summaries on matching card IDs), record `board_synced_at`, reject partial issue/PR failures, and permit a comment-free fallback when optional comment expansion fails. Explicit `pickup` and `board` queries SHALL attempt a bounded remote forge sync when configured (falling back to cache on failure) and surface sync timestamp and fallback status in both `pickup` and `board` responses.

#### Scenario: Worktree PR status labels and stale forge cache caveat

- **WHEN** `WorktreePRStatus` or `BuildPickupSummary` renders worktrees from `.ateam/state.json`
- **THEN** a worktree with no PR card on the board (and not `main`/`master`/`detached` or an ancestor of `main` with `ahead == 0`) is labeled `no PR` rather than `unmerged`
- **AND** when the supplying forge cache timestamp (`SyncedAtFor` for the matched card's provider, or the active forge for `no PR`) is at least `10m` old, PR/forge-derived worktree labels append `(cache <age>)` (e.g. `open PR #64 (cache 4h)`, `no PR (cache 4h)`, `merged via #57 (cache 4h)`), `Worktrees:` prints `labels from forge cache synced <age> ago; run ateam state pickup`, and live git-derived fields (`head`, `dirty`, `ahead`, and `default branch` / `detached` / `merged (ancestor of main)` / `no commits yet`) carry no cache caveat
- **AND** a detached checkout is labeled `detached` on every surface, including after a live head read that names it `HEAD`, and a branch with no PR card whose head equals the `main` head is labeled `no commits yet` rather than `merged (ancestor of main)`

#### Scenario: Open work is never capped

- **WHEN** a forge holds more than 50 open issues or more than 50 open pull requests
- **THEN** synchronization SHALL list every open issue and pull request (GitHub `--state open` with a limit of 1000; Gitea `state=open`, page by page until an empty page or a page that adds no new numbers) and at most the 50 most recently updated closed ones (Gitea closed issues: the 50 newest, since the issues endpoint cannot sort), merged by number with the closed copy winning, so a worktree on an old open pull request's branch is labelled `open PR #<n>` and never `no PR`, and one on a just-merged old pull request is labelled `merged via #<n>` once GitHub's search index holds the merge (until then a sync may briefly label it `no PR`)
- **AND** the list queries for one forge SHALL run concurrently, so the sync fits the 3s per-forge pickup budget
- **AND** the pickup summary's `synced` line names the closed-item cap

### Requirement: Forge token roles

`ateam` SHALL authenticate forge reads (board sync, pickup, review evidence) with `ATEAM_TOKEN_READ` and forge writes (PR comments, label bootstrap) with `ATEAM_TOKEN_WRITE`, each falling back to the other when only one is set, and SHALL keep the existing Gitea (`GITEA_TOKEN`, then `ATEAM_GITEA_TOKEN`) and GitHub (`GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth`) resolution when neither is set. Values are trimmed; a whitespace-only value counts as unset.

#### Scenario: Separate read and write tokens

- **WHEN** `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE` hold different tokens
- **THEN** Gitea board sync sends the read token and a Gitea PR handoff comment or label bootstrap sends the write token, `gh` list calls run with the read token as `GH_TOKEN` and `gh pr comment` / `gh label create` with the write token, and review evidence uses the read token

#### Scenario: One role token serves both roles

- **WHEN** only one of `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE` is set
- **THEN** that token authenticates both reads and writes, ahead of `GITEA_TOKEN` or `GH_TOKEN`

#### Scenario: Doctor warns on a shared token

- **WHEN** `ateam doctor` runs and reads and writes resolve to the same token (neither role variable set, only one set, or both set to one value)
- **THEN** it warns `forge reads and writes share one token; set ATEAM_TOKEN_READ and ATEAM_TOKEN_WRITE to separate tokens scoped to each role`, prints no token value, and `--check` still passes on that warning alone

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

#### Scenario: Stale handoff invalidation on HEAD drift or closed PR

- **WHEN** a recorded handoff's checkpoint SHA no longer matches HEAD, its branch changed, or its subject PR (matched at the start of `goal` or by the handoff's `branch` when no open PR exists for that branch) is closed/merged in a different session
- **THEN** the handoff is marked stale, pickup reports the staleness reason instead of injecting the outdated five-field goal as active work, and pre-compact does not re-save the stale handoff
- **AND** historical PR mentions in `addresses`, `done_when`, or mid-sentence in `goal`, or a fresh handoff written in the current session at the current `HEAD` and `branch`, do not mark the handoff stale

#### Scenario: Worktree merge reconciliation and pickup worktree section

- **WHEN** state is refreshed or `BuildPickupSummary` renders a repository with worktrees
- **THEN** each worktree's `merged_into_main` field is set to `true` when a `done` PR card for its branch has `merged: true` and `head_sha` matching the worktree `head` (with `ahead == 0` ancestor check as fallback and `false` when new commits exist ahead of the merged PR or when an open PR exists), and `BuildPickupSummary` prints a `Worktrees:` section listing each worktree's branch, dirty status, ahead count, and PR status (`merged via #n` or `open PR #n`)

#### Scenario: Per-agent per-branch handoff slots and pickup rendering

- **WHEN** multiple agents or branches record handoffs via `ateam state handoff`, `ateam_state(action="handoff")`, `pre-compact`, or the HTTP API
- **THEN** `.ateam/state.json` stores each handoff in `handoffs["<agent>@<branch>"]` (with `agent` resolved from `--agent`/`agent`, `ATEAM_AGENT`, the active writer lease, or branch prefix) while keeping `last_handoff` aliased to the current session's `<agent>@<branch>` slot
- **AND** `BuildPickupSummary` drops stale entries from `handoffs` per entry, prints the current session's `<agent>@<branch>` handoff first (`Last Handoff:\n...` or `Last Handoff: none for <agent>@<branch>`), and prints one deterministic line per other live handoff (`other handoffs: <agent>@<branch> (<age>, PR #<n>)`)

#### Scenario: A deliberate handoff does not replace another session's fresh handoff

- **WHEN** `ateam state handoff`, `ateam_state(action="handoff")`, or the HTTP API records a handoff into a slot that holds a non-stale handoff which is not a pre-compact auto-save and belongs to a different or unknown session
- **THEN** the write is refused and the slot is unchanged, no PR comment is posted, and the refusal names the slot, the holding session, its goal and its age (CLI exit 2, MCP tool error, HTTP 409)
- **AND** `--force` (CLI), `force: true` (MCP), or `?force=1` (HTTP) replaces it

#### Scenario: Pickup shows handoffs recorded inside linked worktrees

- **WHEN** `BuildPickupSummary` renders a worktree other than the current checkout whose own `.ateam/state.json` holds a handoff for that worktree's branch (`HEAD` for a detached worktree) in a slot the current state does not hold
- **THEN** pickup prints `    handoff: <goal> (<agent>@<branch>, <age>)` under that worktree's line, adding `, stale` when the current state's staleness rules (saved flag, worktree head, closed or merged PR, same-session exception) find it stale
- **AND** a missing or unreadable worktree state prints nothing, and storage of handoffs is unchanged

#### Scenario: State help is read-only

- **WHEN** `ateam state --help`, `ateam state -h`, or `ateam state help` runs
- **THEN** it prints every `state` subcommand and exits 0 without reading or writing `.ateam/state.json`, touching the writer lease, syncing a forge, or posting a comment
- **AND** an unknown `state` subcommand still exits 1 and names `ateam state --help`

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

#### Scenario: Shipped review-pr skill

- **WHEN** a repository is initialized with default templates
- **THEN** `.agents/skills/review-pr/SKILL.md` is written with the same body this repository uses, with thin adapters at `.claude/skills/review-pr/SKILL.md` and `.gemini/commands/review-pr.toml`, and parity checks reject a canonical `review-pr` skill that loses its numbered steps or the record grammar markers `reviewed-by:` and `Reviewed at`

#### Scenario: Build provenance and duplicate server detection in doctor

- **WHEN** `ateam version`, `session-start`, or `ateam doctor` runs
- **THEN** build provenance (`vcs.revision` or `-ldflags` commit) is reported in `ateam version` and the `[ateam]` session-start line (or `unknown` when built without VCS metadata), and `ateam doctor` lists running `ateam serve` and `ateam mcp` processes (with their build commits) and warns when the binary has no build provenance (`binary has no build provenance; build with task build`), when the binary commit is behind checkout `HEAD` (`binary <build> is behind HEAD <head>; <advice>`, where the advice names the binary to rebuild or replace per Requirement: Build provenance names the binary the hooks run) or not an ancestor of checkout `HEAD`, when hook latency exceeds `10000 us` (`<10ms` budget), when a running `ateam mcp` process is stale, or when multiple `ateam serve` processes are running concurrently

#### Scenario: Stale or missing binary build provenance warning at session start

- **WHEN** `ateam hook --event session-start` runs in the ateam checkout (root `go.mod` declares `module github.com/c360studio/ateam`) with a resolved `HEAD` commit
- **THEN** it compares the embedded build commit (`state.BuildRevision()`) against `ReadFastGitHead` without spawning a git subprocess, emits `binary <build> is behind HEAD <head>; <advice>` (the advice names the binary to rebuild or replace per Requirement: Build provenance names the binary the hooks run) when the binary commit differs from `HEAD`, emits `binary has no build provenance; build with task build` when build provenance is `unknown` or `dev`, and emits neither line when the binary commit matches `HEAD`

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

#### Scenario: Forced init replaces a symlinked file instead of writing through it

- **WHEN** `ateam init --force` would replace a generated file whose path is a symbolic link, for example an `AGENTS.md` that links to a shared rules file outside the repository
- **THEN** init removes the link and writes the template as a regular file at that path, so the link's target is never opened for writing and keeps its bytes
- **AND** a symlinked harness document whose target differs from the template is still backed up first, and the backup holds the target's bytes

#### Scenario: Gemini context file uses the nested key

- **WHEN** `ateam init` writes `.gemini/settings.json`, or doctor checks a Gemini settings file
- **THEN** init writes `"context": {"fileName": "AGENTS.md"}` and no top-level `contextFileName`, because Gemini CLI reads only the nested key
- **AND** doctor accepts only a `context.fileName` of `AGENTS.md`; a file that names `AGENTS.md` only in the top-level `contextFileName` fails with `missing context.fileName AGENTS.md (Gemini CLI ignores the top-level contextFileName)`

### Requirement: Board state and authenticated API access

The board SHALL expose state and support card creation, movement, handoff, and mode updates through its API, with the static application served at the root. Without an API token, protected reads and writes SHALL accept only loopback clients; when a token is configured, every client SHALL supply it. Board helpers SHALL group cards into all six canonical columns, filter by text, agent, forge, and column, and format five-field pickup prompts deterministically.

#### Scenario: Repository and dashboard behavior

- **WHEN** a user manages repositories and operates the dashboard
- **THEN** The board SHALL support persisted repository registration, a selected repository, explicit forge settings, canonical OpenSpec requirement details and Current Ops. Mutations SHALL refuse a stale repository selection; failed persistence SHALL not change registration in memory. Explicit forge removal SHALL survive refresh, while automatic detection SHALL follow origin changes. Sync SHALL address the configured target and SHALL not fall back to a different repository after failure. Current Ops SHALL distinguish the selected checkout's inspected dirty state from uninspected worktrees, and compatible checkpoint SHA abbreviations SHALL not imply drift. Dashboard loads SHALL discard superseded results and clear stale state on failure. Dialogs SHALL contain focus, support Escape, and expose submission feedback. Benchmarks SHALL run on demand.

#### Scenario: Repository selection history

- **WHEN** a user adds and selects another repository, then removes it
- **THEN** state and model routing follow that selection and fall back to the startup repository
- **AND** stale clients cannot write to the newly selected repository

#### Scenario: Configured token

- **WHEN** a loopback client requests protected state without the configured token
- **THEN** the request is unauthorized

#### Scenario: Merged worktree badge in Current Ops

- **WHEN** Current Ops renders worktrees alongside synced pull-request cards and git branch ancestry
- **THEN** a worktree whose branch PR is merged (and has no newer unmerged commits ahead of `main` or the merged PR head) or whose branch is an ancestor of `main` with no open PR is marked `merged` (`Merged / potentially stale`) with its merge evidence (`Merged via PR #N` or `Branch ancestor of main`) while preserving local dirty status
- **AND** worktrees with an open PR, a closed-unmerged PR, new commits after a merged PR, or unavailable forge evidence are not marked `merged`

### Requirement: Optional herdr pane identity in the writer lease
When `HERDR_ENV=1`, the writer lease SHALL record `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID` from the environment only, and pickup output SHALL name the pane in the writer line and the different-writer warning. Outside herdr the lease SHALL be byte-for-byte unchanged and SHALL ignore herdr variables.

#### Scenario: Pane variable without herdr
- **WHEN** `HERDR_PANE_ID` is set but `HERDR_ENV` is not `1`
- **THEN** the lease records no herdr identity

#### Scenario: Different writer inside herdr
- **WHEN** a recent lease from another session carries a pane ID
- **THEN** the pickup warning names that pane

### Requirement: Optional herdr caveat notifications
`ateam state herdr on-event` SHALL do nothing unless `HERDR_ENV=1` and `HERDR_BIN_PATH` are set. On `worktree.created` and non-working `pane.agent_status_changed` events in an ateam repository, it SHALL toast each `HALT`, `RED`, or `BLOCKED` caveat at most once per repository. A failed toast SHALL NOT prevent the remaining claimed toasts from being attempted. It SHALL never run on the synchronous hook path.

#### Scenario: Gate closed
- **WHEN** `HERDR_ENV` is unset
- **THEN** herdr is never invoked

#### Scenario: Repeat and concurrent events
- **WHEN** the same caveat is seen by repeated or concurrent events, or from a linked worktree
- **THEN** exactly one toast is shown
- **AND** identity SHALL be the change, marker, and full caveat text, shared across path spellings and linked worktrees of one git common dir
- **AND** the record SHALL be saved under a file lock before toasting

#### Scenario: Partial delivery failure
- **WHEN** the first of two claimed toasts fails
- **THEN** the second is still attempted and the error is reported

### Requirement: herdr plugin panes
`ateam state herdr open-pane` SHALL accept only the `pickup` and `spec-queue` panes and open them through `herdr plugin pane open`. It SHALL pass the invoking pane's cwd and SHALL NOT pass a workspace, which herdr rejects for popups. Outside herdr it SHALL fail without invoking herdr.

#### Scenario: Action from a pane
- **WHEN** the action runs with `HERDR_PANE_ID` set
- **THEN** the popup opens with that pane's cwd

### Requirement: Accurate lab positioning and adoption boundaries

The README and adoption documentation SHALL identify ATeam as an opinionated lab primarily grounded in Go development. They SHALL distinguish language-independent coordination mechanisms from Go-specific tooling, project-owned verification tasks, and unvalidated language support. Relevant generated instructions SHALL reflect the same limitations.

#### Scenario: Adopting in a non-Go repository

- **WHEN** a reader evaluates ATeam for a non-Go project
- **THEN** the documentation explains that initialization does not detect or adapt to the project language, mutation tooling requires non-test Go source, shortcut scanning uses an explicit extension allowlist, and the project must supply its own Taskfile and verification commands

#### Scenario: Interpreting benchmark and discipline claims

- **WHEN** a reader evaluates Ponytail modes and benchmark results
- **THEN** the documentation distinguishes injected agent guidance from enforced behavior and describes reference-fixture self-tests without presenting them as proof of improved agent performance

### Requirement: Deterministic test-to-spec attribution

ATeam SHALL enforce attached machine-readable requirement citations for supported Go and Vitest tests through an offline deterministic gate in `task verify`. Missing, malformed, unsafe, or unresolved citations and parse failures SHALL fail verification. Attribution SHALL NOT be presented as semantic coverage proof. Detailed discovery, syntax, and resolution rules are specified by the test-attribution capability.

#### Scenario: Valid multiple requirement citations

- **WHEN** a supported test has attached valid citations to two unique requirements within the repository
- **THEN** the checker accepts that test's attribution and produces deterministic results

#### Scenario: Missing or unresolved attribution

- **WHEN** a supported test has no citation or cites a missing or ambiguous requirement
- **THEN** the checker reports the test or citation location and exits nonzero

#### Scenario: Unsafe reference or malformed source

- **WHEN** a citation escapes the repository or a supported source file cannot be parsed
- **THEN** the checker refuses the input without accepting incomplete scan results

#### Scenario: Verification integration

- **WHEN** task verify runs on a tree with an unattributed supported test
- **THEN** verification fails on the attribution gate

### Requirement: Client-readable MCP registration

ATeam initialization SHALL place MCP registration in the supported configuration surface for each supported client, preserve unrelated existing settings, and document any required migration or manual setup. Local parity checks SHALL validate the expected registration structure without representing that result as proof of a live client connection.

#### Scenario: Codex MCP discovery

- **WHEN** ATeam is initialized for Codex in a trusted disposable project with its executable available
- **THEN** the registration uses Codex's supported MCP configuration surface and client discovery can identify the ATeam server without relying on an entry in a hooks file

#### Scenario: Existing client configuration

- **WHEN** initialization encounters existing client settings and servers
- **THEN** unrelated configuration is preserved and any required manual migration is explained instead of silently overwriting settings

### Requirement: Accurate MCP integration documentation

The MCP guide SHALL document the implemented tools and actions, prompts, resources, launch requirements, repository working directory, state/task/posting side effects, and verification steps. It SHALL distinguish local configuration checks, direct protocol tests, and real-client connection evidence.

#### Scenario: Following the MCP guide

- **WHEN** a user follows the setup and smoke-check instructions
- **THEN** examples match the exposed schema, do not advertise an unsupported sync action, and explain local-only handoff using post=false

#### Scenario: Posting side effect at the tool boundary

- **WHEN** an MCP client lists tools
- **THEN** the `ateam_state` `post` property states that an omitted or `true` value attempts a PR comment and `false` saves the handoff locally only, and the canonical handoff skill tells agents to pass `post` explicitly

### Requirement: Default client context compaction limits

ATeam initialization SHALL scaffold the context and compaction defaults specified below into newly created client configurations while preserving pre-existing client files unchanged. Parity checks SHALL validate configured fields, reject out-of-range or non-numeric values, and permit valid custom or omitted fields in preserved configurations.

#### Scenario: Newly initialized repository context limits

- **WHEN** `ateam init` creates `.claude/settings.json`, `.codex/config.toml`, and `.gemini/settings.json` in a repository without pre-existing client configs
- **THEN** `.claude/settings.json` sets `autoCompactWindow` to `500000`, `.codex/config.toml` sets top-level `model_context_window = 500000` and `model_auto_compact_token_limit = 450000`, and `.gemini/settings.json` sets `model.chatCompression.contextPercentageThreshold` to `0.5`

#### Scenario: Invalid configured context limit rejected by parity check

- **WHEN** a project client configuration specifies a non-positive or out-of-range context limit (such as `autoCompactWindow < 100000` or `> 1000000`, `model_auto_compact_token_limit > model_context_window`, or `contextPercentageThreshold` outside `(0, 1]`)
- **THEN** `CheckParity` reports a violation naming the file and field

### Requirement: Merge gate and CI workflow enforcement

CI SHALL enforce the pinned verification workflow and aggregate `Required` check. The merge check SHALL require an active strict `main` ruleset (on Gitea, branch protection requiring `CI / Required (pull_request)` with `block_on_outdated_branch`), review by a not-weaker reviewer for non-draft code PRs, and no open known flakes unless the PR closes them, on GitHub and on Gitea.

#### Scenario: Complete workflow and merge-check contract

- **WHEN** the workflow and merge check are evaluated
- **THEN** `.github/workflows/ci.yml` SHALL trigger on `push` to `main` and `pull_request` (`opened`, `synchronize`, `reopened`, `ready_for_review`), pin every job to `ubuntu-24.04` with read-only permissions, run `Verify` (`15` minute timeout, `ateam doctor --check` and `task verify`) and `Merge check` (`scripts/merge-check.sh`), and gate merges on the aggregate `Required` job (`if: always()`, reading `join(needs.*.result, ' ')`). `scripts/merge-check.sh` (`task merge:check`) SHALL verify the active `main` ruleset requiring `Required` (`strict_required_status_checks_policy=true`), classify a changed file (and any `previous_filename` on rename) as a document only when its path ends in `.md` (outside `.claude/agents/` and `.agents/contracts/`) or starts with `openspec/` and ends in `.md`, `.yml`, `.yaml`, or `.json`, enforce review (`implemented-by:` and `reviewed-by:` each naming one of `claude`, `codex`, `gemini`, the reviewer not weaker by tier then band through `ateam review compare`, and different agents only when the policy sets `cross_provider`) on non-draft code pull requests, and block merges while any open `class:flake` issue is not closed by the pull request.

#### Scenario: Pinned CI workflow and Required aggregate

- **WHEN** `.github/workflows/ci.yml` is evaluated against the merge-gate contract
- **THEN** `pull_request` lists `opened`, `synchronize`, `reopened`, and `ready_for_review`, every job runs on `ubuntu-24.04` without write permissions, `merge-check` has only `contents: read`, `issues: read`, and `pull-requests: read`, `verify` and `merge-check` run under `if: github.event_name != 'pull_request' || !github.event.pull_request.draft`, `required` needs `verify` and `merge-check` under `if: always() && (github.event_name != 'pull_request' || !github.event.pull_request.draft)` and fails unless every needed job succeeded, each of the three is named `${{ github.event.pull_request.draft && '<name> (draft, skipped)' || '<name>' }}` for its check name `Verify`, `Merge check`, or `Required`, and `concurrency.cancel-in-progress` is `${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}` so the run that marking a draft ready starts never cancels the in-flight run for the same head and leaves a failed `Required` on it

#### Scenario: Review strength and known-flake merge check

- **WHEN** `scripts/merge-check.sh` runs for a pull request or a `main` push
- **THEN** it checks the active `main` ruleset for `Required` (`strict_required_status_checks_policy=true`), treats any path under `openspec/` not ending in `.md`, `.yml`, `.yaml`, or `.json` (such as `.py`, `.sh`, `.go`, `.lock`, or executables) as code, requires non-draft code PRs to carry one `implemented-by:` and one `reviewed-by:` line each naming an agent (`claude`, `codex`, `gemini`) and its model in parentheses after the agent (several comma-separated on `implemented-by:`; a missing or blank model is a finding); once `.agents/review-policy.json` exists, each model is followed by its native effort (`claude (sonnet, medium, opus, high)`), the runs pass to `$ATEAM review compare` (default `ateam`; the CI job and `task merge:check` build it) whose findings, including a missing effort, are review findings and whose exit other than 0 or 1 is unavailable (exit 2), and the two lines must name different agents only when the policy sets `cross_provider: true`; verifies PR commit list completeness against the PR's `commits` count and `head.sha`, requires a PR comment review record (`reviewed-by: <agent>` or a `### Independent Cross-Agent Review` heading) designating a single full or short commit SHA in the PR's commit list, paginates commit file lists (failing closed on truncated pages or the 3,000-file ceiling) to allow later commits after the latest recorded commit only when each is a merge of `main` touching none of the PR's changed files or a documents-only commit (failing any other later commit with `review record at <sha> is older than content commit <sha>; request re-review`), and blocks merges when any open `class:flake` issue is not closed by the pull request

#### Scenario: Gate-file changes wait for the owner

- **WHEN** a pull request's changed files, or a rename's previous path, include `.agents/review-policy.json`, `scripts/merge-check.sh`, `scripts/gate-files.sh`, anything under `.github/workflows/` or `.gitea/workflows/`, or anything under `internal/review/`
- **THEN** the `Gate files` check (`.github/workflows/gate-files.yml` on `pull_request_target`, running `scripts/gate-files.sh` from the base branch with read-only permissions and never checking out or running pull request code) fails until the pull request carries the label `gate-change-approved`, and passes for a pull request that touches no gate file
- **AND** a failed read, a file list that is not a list of file objects, or one shorter than the pull request's `changed_files` (the listing stops at 3,000 files) exits 2 naming the read, never counted as no gate file
- **AND** the same holds on Gitea for `.gitea/workflows/gate-files.yml`, which runs `scripts/gate-files.sh` with `ATEAM_FORGE=gitea` to read Gitea's API with `curl`, every file page until an empty one, and the labels from the pull request
- **AND** the `Merge check` job checks out and builds from the base commit (`github.event.pull_request.base.sha`, or `github.sha` on a push), so the merge-check script, the review policy it reads, and the `ateam` binary are never taken from the pull request they judge

#### Scenario: Merge check on either forge

- **WHEN** `scripts/merge-check.sh` runs
- **THEN** it runs `$ATEAM review gate` (default `ateam`), which picks the forge from `ATEAM_FORGE`, else under Actions from `GITHUB_SERVER_URL` (`https://github.com` is GitHub, any other root is Gitea) with `GITHUB_REPOSITORY`, else from `origin`, else GitHub through `gh`, and prints `merge-check: forge=<forge> repo=<owner>/<repo> (from <source>)` without the host
- **AND** GitHub is read through `gh` with the argv the shell script used, and Gitea through API v1 with `GITEA_TOKEN`, with the same review and known-flake lines on both
- **AND** `.gitea/workflows/ci.yml` is `.github/workflows/ci.yml` with only the merge-check step's credentials changed, to `GITEA_TOKEN: ${{ github.token }}` (no extra secret)

#### Scenario: Gitea up-to-date rule

- **WHEN** the merge check reads Gitea
- **THEN** the up-to-date rule holds only when `main` is `protected`, the branch protection in force on it (`effective_branch_protection_name`) has `enable_status_check` and lists exactly `CI / Required (pull_request)` in `status_check_contexts` (no glob), and, when the rule can be read, has `block_on_outdated_branch`; otherwise it fails like a GitHub ruleset without the strict setting
- **AND** when reading the rule answers 403 or 404 (the Gitea Actions job token, the only token the CI job uses), the branch's `protected`, `enable_status_check` and `status_check_contexts` decide and the run prints `merge-check: rule strict=unverified (branch protection needs a repo-admin token; the job token can see protected, status check and context only)`
- **AND** any other failed read of the branch or the rule (401, 5xx, a wrong shape) exits 2, never passes

#### Scenario: Gitea review evidence

- **WHEN** the merge check reads a Gitea pull request
- **THEN** its file and commit lists are read page by page and must match `X-Total-Count` (and the files `changed_files`), the commit list is taken oldest first and must end at `head.sha`, and every commit after the review record is a finding needing a new review until a live run shows Gitea's commit file lists are complete

#### Scenario: Gitea closing references

- **WHEN** an open `class:flake` issue exists on Gitea
- **THEN** the label is read from the repository's labels first (Gitea discards an unknown label in an issue filter, so a missing label never reads as an empty list), the open issues are filtered for the label client side, and a flake counts as closed only when its timeline holds a `ref_action` of `closes` from this pull request in the same repository; the description is not parsed

#### Scenario: Draft pull requests defer CI and ready revisions run every gate

- **WHEN** a `pull_request` or `pull_request_target` event arrives on GitHub for a draft pull request
- **THEN** `Verify`, `Merge check`, `Required`, and `Gate files` do not run (`if: github.event_name != 'pull_request' || !github.event.pull_request.draft`, with `Required` as `always() && (...)`, and `Gate files`, whose event is always `pull_request_target`, as `if: ${{ !github.event.pull_request.draft || github.event.action == 'unlabeled' }}`, so removing a label on a draft still replaces an earlier `Gate files` pass at the same head), and each is renamed on a draft (`name: ${{ github.event.pull_request.draft && '<job> (draft, skipped)' || '<job>' }}`) so no check run named `Required` or `Gate files` exists at a draft head and the required contexts stay unreported; GitHub does not evaluate a skipped job's `name:`, so the draft's skipped check run carries the raw expression text, which is also not a required name
- **AND** a non-draft `opened` or `reopened`, a `ready_for_review`, and every `synchronize` while the pull request is not a draft run `Verify`, `Merge check`, `Required`, and `Gate files` against the head revision under their required names, and `push` to `main` still runs `Verify`
- **AND** whenever `Required` runs for a non-draft revision it fails on any needed result other than `success`, including `skipped`
- **AND** the Gitea twins do not defer drafts: `.gitea/workflows/ci.yml` keeps plain job names, no draft guard, and `required` under exactly `if: always()`, and `.gitea/workflows/gate-files.yml` keeps the plain name and no guard, because Gitea 1.22 evaluates neither `name:` nor a compound `always()`, reports a skipped job as success, and emits `label_updated` for label changes; those lines are the only further differences from their GitHub twins

### Requirement: MCP process attribution

`ateam doctor` and the session-start hook SHALL warn only about `ateam mcp` processes that serve the current repository, and SHALL compare a server's build with checkout `HEAD` only in the ateam checkout.

#### Scenario: Foreign MCP process

- **WHEN** a live `ateam mcp` process is not in this root's registry and its working directory is readable and is neither this root nor a linked worktree of it
- **THEN** doctor lists it in `running_mcp_processes` with `repo: other` and emits no warning for it, and the session-start hook prints no line for it

#### Scenario: Unregistered MCP process in this repository

- **WHEN** a live `ateam mcp` process is not in this root's registry and its working directory is this root, or its working directory cannot be read
- **THEN** doctor and the hook warn about it as before

#### Scenario: Older-than-HEAD comparison outside the ateam checkout

- **WHEN** doctor or the session-start hook runs in a checkout whose root `go.mod` does not declare `module github.com/c360studio/ateam`
- **THEN** the hook prints no `MCP server PID n (build x) is older than HEAD; restart the client` line
- **AND** doctor does not compare a server's build with checkout `HEAD`; it compares with the on-disk `bin/ateam` build, as it does when there is no `HEAD`

### Requirement: Adopter rules parity

`ateam doctor` SHALL check an adopter's rule files for agreement with each other, and its protocol for the markers the harness depends on, without requiring ateam's own section titles or protocol headings.

#### Scenario: Adopter rule sections that agree

- **WHEN** `AGENTS.md` and `.agents/rules-index.md` have the same numbered rule sections, with the same titles in the same order and the same row count per section, whatever the titles, their number, or the heading level used in each file (per file, the one heading level whose numbered sections own table rows; other numbered headings are ignored), and whatever the table's header wording, separator alignment (a separator row has at least one `-`; an all-blank row is a row), indentation or spacing after `|`
- **THEN** doctor reports no rules-parity violation

#### Scenario: Ambiguous rule heading levels

- **WHEN** numbered `## N.` and numbered `### N.` sections in the same file both own table rows
- **THEN** doctor reports a violation for that file saying the rule sections are ambiguous and that only the rule section headings should be numbered, and does not certify parity by picking one level

#### Scenario: Adopter rule sections that disagree

- **WHEN** a numbered section's title, position, or row count differs between `AGENTS.md` and `.agents/rules-index.md`
- **THEN** doctor reports a violation naming the section and the file in which it differs

#### Scenario: Adopter rule section without table rows

- **WHEN** a numbered rule section in `AGENTS.md` or `.agents/rules-index.md` has no table rows, even when the other file agrees
- **THEN** doctor reports a violation naming the section and the file

#### Scenario: Protocol markers

- **WHEN** `.agents/protocol.md` lacks any of `Deliverable`, `non-goals`, `Definition of Done`, `Stop boundary`, `ateam state handoff`, or `openspec/changes/archive/`
- **THEN** doctor reports a violation naming the missing obligation and where the upstream template puts it, and its headings are not checked

### Requirement: Adopter skills and adapter aliases

`ateam doctor` SHALL apply its canonical-skill checks only to the five skills ateam generates (`pickup`, `handoff`, `preflight`, `ponytail`, `review-pr`), and SHALL check other skills and adapters only for links that resolve.

#### Scenario: Adopter skill

- **WHEN** `.agents/skills/` holds a skill that is not one of the five canonical skills and that no canonical adapter aliases
- **THEN** doctor requires no numbered heading, procedure markers, adapters, line cap or checkbox rule for it, and lists it under `adopter_skills` in its JSON output (an empty list when there is none)

#### Scenario: Dangling adapter link

- **WHEN** a `.claude/skills/<x>/SKILL.md` or `.gemini/commands/<x>.toml` adapter links to a path under `.agents/skills/` that is not a regular file (a link naming only a skill means its `SKILL.md`; a link starting `./` or `../` resolves from the adapter's own directory, a bare one from the repository root)
- **THEN** doctor reports a violation naming the adapter and the missing target (`<adapter>: links to missing <target>`), and does not report the link as an alias

#### Scenario: Adapter alias

- **WHEN** an adapter named `<x>` links to an existing `.agents/skills/<y>/SKILL.md` with `<y>` different from `<x>` (an adapter's skill is the one its first `.agents/skills/` link names, and a trailing `.` is not part of the name)
- **THEN** doctor lists `alias: .claude/skills/<x> -> .agents/skills/<y>` (or `alias: .gemini/commands/<x>.toml -> .agents/skills/<y>`) under `skill_aliases` in its JSON output without a violation, and when `<x>` is a canonical skill its canonical checks read `.agents/skills/<y>/SKILL.md` and also `.agents/skills/<x>/SKILL.md` (Codex and Antigravity read `.agents/skills/` directly, so the generic file is checked even when both adapters alias elsewhere), and its adapter must link to `<y>` and say "fully"

#### Scenario: Canonical adapters resolving to different skills

- **WHEN** the Claude and Gemini adapters of a canonical skill resolve to different skills, for example the Claude adapter aliases `<y>` and the Gemini adapter links to the generic `<x>`
- **THEN** doctor applies that canonical skill's checks to each resolved `SKILL.md` and to the generic `.agents/skills/<x>/SKILL.md`, and a canonical adapter that is missing or has no `.agents/skills/` link must link to the generic file as before

### Requirement: Hooks registered outside the project settings

`ateam doctor` SHALL accept ateam's hooks registered outside the project's client settings files only when `ATEAM_HOOK_SCOPE=user` is set, and SHALL report which file satisfied each client's hook check under `hook_sources` in its JSON output (an object keyed by client agent, `{}` when none passed).

#### Scenario: Claude hooks at user scope

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and `.claude/settings.json` has no ateam hooks, but `.claude/settings.local.json` or `~/.claude/settings.json` (read in that order) carries every required `ateam hook --event <event>` command and `--agent claude`
- **THEN** doctor reports no Claude hook violation and names the file that carried the hooks; Codex reads `~/.codex/hooks.json` and Gemini `~/.gemini/settings.json` the same way

#### Scenario: Partial, malformed or split local or user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set, the project file lacks the hooks, and no single local or user-scope file is valid JSON carrying every required event and the client's `--agent` (hooks split across files count as missing)
- **THEN** doctor reports the project file's violations and each local or user-scope file's, for example `~/.claude/settings.json: invalid JSON` or `~/.claude/settings.json: missing hook command for event <event>`

#### Scenario: Knob unset

- **WHEN** `ATEAM_HOOK_SCOPE` is unset and `.claude/settings.json` has no ateam hooks
- **THEN** doctor reports `missing hook command for event <event>` for `.claude/settings.json`, as before, whatever the local and user-scope files contain

#### Scenario: Gemini context.fileName

- **WHEN** `.gemini/settings.json` sets its context file (the nested `context.fileName`; the top-level `contextFileName` is not read) to anything other than `AGENTS.md`, or sets none while either the knob is unset or `~/.gemini/settings.json` is missing, invalid or lacks `context.fileName: "AGENTS.md"`
- **THEN** doctor reports `<file>: missing context.fileName AGENTS.md` naming the file whose value was checked (the project file when it sets a value, as Gemini gives project settings precedence; `~/.gemini/settings.json` when the knob is set, the project file sets none, and that file parses), whichever file carried the hooks

#### Scenario: Unknown knob value

- **WHEN** `ATEAM_HOOK_SCOPE` is set to anything other than `user` (compared case-insensitively, surrounding space ignored)
- **THEN** doctor reports `ATEAM_HOOK_SCOPE=<value>: unknown value; set it to user or unset it` and reads only the project files

#### Scenario: Client without user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and a client's project hook file lacks the ateam hooks and doctor reads no user-scope hook file for that client (Antigravity's `.agents/hooks.json`)
- **THEN** doctor reports that the client has no user-scope hooks doctor reads (`.agents/hooks.json: ATEAM_HOOK_SCOPE=user, but agy has no user-scope hooks doctor reads`), rather than passing

#### Scenario: Hook commands match exactly under their native event

- **WHEN** a hook file's only command for a required event names a different event (`--event turn-ended` for `turn-end`), sits under a native event key other than the one that maps to that event (a `turn-end` command under `SessionStart`), or carries the client's `--agent` only outside a hook command (a permissions entry `Bash(ateam hook --agent claude:*)` while the hooks say `--agent codex`)
- **THEN** doctor reports `missing hook command for event <event>` or `missing --agent <client> in hook commands` for that file and does not name it under `hook_sources`, with or without `ATEAM_HOOK_SCOPE=user`

### Requirement: Merge check notes an unapproved OpenSpec change

On a code pull request, the merge check SHALL print a note for each OpenSpec change directory the pull request touches whose `proposal.md` at the head lacks a filled `Approved-by:` header, and the note SHALL NOT change the exit status.

#### Scenario: Unapproved change on a code pull request

- **WHEN** a code pull request adds files under `openspec/changes/foo/` and `openspec/changes/foo/proposal.md` at its head has `Approved-by:` with no value
- **THEN** the merge check prints `merge-check: note: OpenSpec change foo has no Approved-by; the spec was not human-reviewed` and its exit status is what it would be without the note

#### Scenario: Approved change

- **WHEN** the touched change's `proposal.md` has `Approved-by: Owner (2026-10-26)`
- **THEN** the merge check prints no note for it

#### Scenario: Archived change and unreadable proposal

- **WHEN** the pull request touches `openspec/changes/archive/2026-10-26-foo/` whose `proposal.md` is missing, or the read of a touched `proposal.md` fails
- **THEN** the merge check prints a note naming the change or the path, and a failed read does not make the check unavailable

### Requirement: Merge check requires an approving review verdict

The merge check SHALL accept the latest review record only when it carries exactly one `Verdict: APPROVE` line outside fenced code blocks, and SHALL ignore fenced code blocks when it identifies a review record and its designated commit.

#### Scenario: Rejecting record blocks

- **WHEN** the latest review record names the reviewer and one commit and says `Verdict: REQUEST CHANGES`
- **THEN** the merge check reports a blocking finding naming that verdict and exits non-zero

#### Scenario: Missing or repeated verdict blocks

- **WHEN** the latest review record has no unfenced `Verdict:` line, or more than one
- **THEN** the merge check reports a blocking finding naming what it found and exits non-zero

#### Scenario: Later approval supersedes

- **WHEN** a `Verdict: REQUEST CHANGES` record is followed by a record from the same reviewer with `Verdict: APPROVE` naming the head
- **THEN** the merge check accepts the review record

#### Scenario: Fenced text is not a record

- **WHEN** a comment's only `reviewed-by:`, `Reviewed at`, or `Verdict:` line is inside a ``` or ~~~ fence
- **THEN** the merge check does not count that line

### Requirement: Opt-in design track

A non-trivial change SHALL declare the `quick` or `design` track at Clarify. `ateam state spec-check` SHALL warn, without failing, when a change's `design.md` exceeds 120 lines. `ateam init` SHALL write the `design` canonical skill with its Claude and Gemini adapters, and `ateam doctor --check` SHALL fail when the canonical `design` skill is missing.

#### Scenario: Design over the length limit

- **WHEN** a change's `design.md` has 121 or more lines
- **THEN** `ateam state spec-check` prints a warning naming `openspec/changes/<id>/design.md`, its line count, and the limit of 120, and still exits 0; at 120 lines, or with no `design.md`, it prints no such warning

#### Scenario: Design skill parity

- **WHEN** `ateam init` runs on an empty directory
- **THEN** it writes `.agents/skills/design/SKILL.md`, `.claude/skills/design/SKILL.md`, and `.gemini/commands/design.toml`
- **AND** `ateam doctor --check` fails once `.agents/skills/design/SKILL.md` is removed

### Requirement: Default-branch handoff freshness follows the remote tip

For any session other than its holder's, a handoff in a default-branch slot SHALL be stale when its checkpoint differs from the default branch's remote-tracking tip already on disk, unless the checkpoint is the local HEAD and the local branch is ahead of that tip. A deliberate handoff written with no resolvable session id SHALL warn, and a slot refusal SHALL name the holder's writer-lease state when it is known.

#### Scenario: Checkpoint behind origin does not hold the slot

- **WHEN** `claude@main` holds a handoff at checkpoint `A`, local `main` is at `A`, and `origin/main` is at a later commit `B`
- **THEN** the handoff is stale with reason `origin/main moved from A -> B` and a new handoff from another session is written

#### Scenario: Unpushed local main keeps its handoff fresh

- **WHEN** the handoff's checkpoint is local `main`'s HEAD and local `main` is ahead of `origin/main`
- **THEN** the handoff is fresh and still refuses another session's write

#### Scenario: Holder's own session keeps its handoff across compaction

- **WHEN** the holder's session compacts or starts again with local `main` unpulled and `origin/main` ahead of the checkpoint
- **THEN** the handoff keeps its five fields and is shown at session start, while another session can still replace it

#### Scenario: Handoff without a session id warns

- **WHEN** a deliberate handoff is written and no session id resolves
- **THEN** the writer is told the handoff will hold its slot against every session

#### Scenario: Refusal names the holder's writer lease

- **WHEN** a handoff is refused because another session's fresh handoff holds the slot and the writer lease belongs to that session
- **THEN** the refusal says whether that lease is live, with its pid, or exited; it says nothing about a lease it cannot attribute

### Requirement: Read another agent's handoff without changing ownership

Pickup SHALL show an attributed, bounded goal preview for every other handoff slot it lists, and `ateam state read-handoff <agent>@<branch>` and `ateam_state(action="read_handoff", slot=...)` SHALL print one named slot's five fields without writing `.ateam/state.json`, touching the writer lease, changing the caller's agent or session, or posting to a forge.

#### Scenario: Pickup previews another agent's goal

- **WHEN** pickup runs as `codex` on `main` and the cache holds a live `claude@main` handoff
- **THEN** pickup prints `Last Handoff: none for codex@main` and `other handoffs: claude@main (<age>[, PR #<n>]): <goal>`, the goal clipped to 104 bytes on a word boundary with `…`

#### Scenario: Reading a named slot is read-only

- **WHEN** a Codex session runs `ateam state read-handoff claude@main` or `ateam_state(action="read_handoff", slot="claude@main")`
- **THEN** it prints the slot's five-field prompt, looking in the current checkout's cache and then linked worktrees' caches
- **AND** the cache bytes, the writer lease, and the caller's resolved agent and session are unchanged, and no forge call is made

#### Scenario: Stale and missing slots

- **WHEN** the named slot is stale under the handoff staleness rules
- **THEN** its fields are printed under `STALE — <reasons>` and the command exits 0
- **AND WHEN** no cache holds the named slot, the command exits 1 (MCP tool error) and lists the slot keys that exist

### Requirement: Prune finished worktrees the caller owns

`ateam state prune [--dry-run] [--agent <name>]` SHALL remove a linked worktree, its local branch, and its remote branch only when the caller's agent owns the worktree, its tree is clean, and the refreshed state labels it finished. It SHALL list every other linked worktree with the reason it was kept. Pickup SHALL mark each worktree the same check would remove.

#### Scenario: A finished, clean, owned worktree is removed

- **WHEN** `claude` runs `ateam state prune` and a linked worktree on `claude/<topic>` is clean and labelled `merged via #<n>` or `merged (ancestor of main)`
- **THEN** the worktree is removed only if its live branch and head still match the refreshed state, the local branch is deleted only while it points at that head, the remote branch is deleted only under a lease on that head (`--force-with-lease`), and one `removed <branch>: <label>` line is printed
- **AND** a worktree whose head or branch moved since the refresh is kept as `moved since refresh`, a local or remote branch that moved before its deletion is kept and named, and a branch deletion that fails without the branch having moved is reported as `failed` with git's error

#### Scenario: Unsafe or foreign worktrees are kept

- **WHEN** a linked worktree is dirty, has an unknown dirty state, is labelled `ahead of merged`, `open PR`, `no PR`, or `no commits yet`, is on another agent's prefix or no agent prefix, is the current checkout, or is on the default branch
- **THEN** it is listed as `kept <branch>: <reason>` and nothing about it changes

#### Scenario: A detached review checkout of a closed PR is removed

- **WHEN** a detached linked worktree's directory is named `<agent>-review-pr-<n>[-<suffix>]` for the caller's agent, no cached PR `<n>` on any forge is open, and its head equals the head of a closed or merged cached PR `<n>`
- **THEN** the checkout is pinned to a temporary branch, removed, and the pin deleted only while it still points at that head, so a commit landing during removal stays on the pin
- **AND** if any cached PR `<n>` is open, or no closed PR `<n>` has that head, the worktree is kept

#### Scenario: Dry run and pickup hint

- **WHEN** `--dry-run` is passed
- **THEN** the same lines are printed with `would remove` in place of `removed`, and no worktree, branch, or remote ref changes
- **AND WHEN** pickup renders a worktree the same check would remove for the current agent, its line ends with `, prune candidate)`

### Requirement: Build provenance names the binary the hooks run

`ateam version`, `ateam doctor`, and the session-start hook SHALL label a binary installed with `go install <module>@<tag>` by its tag, never `unknown`. Doctor SHALL report the `PATH` binary the hooks run as `hook_binary` beside `disk_build`. In the ateam checkout, every stale-binary warning SHALL name the binary's path and the commands that replace it. Pickup SHALL report how many commits local `main` trails `origin/main`.

#### Scenario: Module install reads as its tag

- **WHEN** the binary carries main-module version `v0.3.3` and no stamped commit or `vcs.revision`
- **THEN** `ateam version` prints `ateam v0.3.3 (module)`, and the session-start hook in the ateam checkout names `v0.3.3` instead of saying the binary has no build provenance

#### Scenario: Hook binary differs from the checkout build

- **WHEN** doctor runs in the ateam checkout and the `ateam` on `PATH` reports a different build from `./bin/ateam`
- **THEN** `hook_binary` carries the `PATH` binary's path, version, and build, and a warning names both binaries and how to replace the `PATH` one

#### Scenario: Hook binary matches

- **WHEN** the `PATH` `ateam` and `./bin/ateam` report the same build
- **THEN** doctor emits no hook-binary warning

#### Scenario: Local main behind origin

- **WHEN** local `main` trails `origin/main` by 2 commits
- **THEN** pickup's `main` worktree line and the `[ateam]` summary show `behind=2`, and neither shows `behind` when the two are even

