## MODIFIED Requirements

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
