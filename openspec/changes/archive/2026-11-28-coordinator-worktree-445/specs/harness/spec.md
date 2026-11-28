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

#### Scenario: Pickup shows handoffs recorded inside linked worktrees

- **WHEN** `BuildPickupSummary` renders a worktree other than the current checkout whose own `.ateam/state.json` holds a handoff for that worktree's branch (`HEAD` for a detached worktree) in a slot the current state does not hold
- **THEN** pickup prints `    handoff: <goal> (<agent>@<branch>, <age>)` under that worktree's line, adding `, stale` when the current state's staleness rules (saved flag, worktree head, closed or merged PR, same-session exception) find it stale
- **AND** a missing or unreadable worktree state prints nothing, and storage of handoffs is unchanged

#### Scenario: State help is read-only

- **WHEN** `ateam state --help`, `ateam state -h`, or `ateam state help` runs
- **THEN** it prints every `state` subcommand and exits 0 without reading or writing `.ateam/state.json`, touching the writer lease, syncing a forge, or posting a comment
- **AND** an unknown `state` subcommand still exits 1 and names `ateam state --help`

#### Scenario: A handoff post whose outcome is unknown

- **WHEN** `ateam state handoff` or `ateam_state(action="handoff")` posts the PR comment and the send times out
- **THEN** the CLI prints `forge PR comment uncertain: <err>; rerun ateam state handoff to post it once` (exit 0) and MCP prints `PR comment: uncertain (<err>)`, never `skipped`, and no commit status is posted
- **AND WHEN** a later post finds that the PR's newest handoff comment has the same body, it posts nothing and reports `posted (already on PR)`, while a different newest body, or a failed read of the comments, posts

#### Scenario: A handoff on a branch with no open PR

- **WHEN** `ateam state handoff` or `ateam_state(action="handoff")` runs on GitHub or Gitea for a branch with no open pull request, including one whose only pull request is merged or closed
- **THEN** no comment is posted on any pull request, the CLI prints no `forge PR comment` line (exit 0), MCP prints `PR comment: skipped (no open PR for branch)`, and no commit status is posted
- **AND WHEN** the read that looks for the open pull request fails, nothing is posted and the CLI prints `forge PR comment skipped: <err>`

### Requirement: Prune finished worktrees the caller owns

`ateam state prune [--dry-run] [--agent <name>] [--force]` SHALL remove a linked worktree, its local branch, and its remote branch only when the caller's agent owns the worktree, its tree is clean, the refreshed state labels it finished, and, unless `--force` is passed, no other session occupies it. It SHALL list every other linked worktree with its keep reason, print a warning when forge sync fell back to the cache, and mark prune candidates in pickup.

#### Scenario: A finished, clean, owned worktree is removed

- **WHEN** `claude` runs `ateam state prune` and a linked worktree on `claude/<topic>` is clean and labelled `merged via #<n>` or `merged (ancestor of main)`
- **THEN** the worktree is removed only if its live branch and head still match the refreshed state, the local branch is deleted only while it points at that head, the remote branch is deleted only under a lease on that head (`--force-with-lease`), and one `removed <branch>: <label>` line is printed
- **AND** a worktree whose head or branch moved since the refresh is kept as `moved since refresh`, a local or remote branch that moved before its deletion is kept and named, and a branch deletion that fails without the branch having moved is reported as `failed` with git's error

#### Scenario: Unsafe or foreign worktrees are kept

- **WHEN** a linked worktree is dirty, has an unknown dirty state, is labelled `ahead of merged`, `open PR`, `no PR`, `PR unknown (cache predates branch)`, or `no commits yet`, is on another agent's prefix or no agent prefix, is the current checkout, or is on the default branch
- **THEN** it is listed as `kept <branch>: <reason>` and nothing about it changes
- **AND** a worktree on a branch with no agent prefix is kept as `no agent prefix (host-created branch)`

#### Scenario: A detached review checkout of a closed PR is removed

- **WHEN** a detached linked worktree's directory is named `<agent>-review-pr-<n>[-<suffix>]` for the caller's agent, no cached PR `<n>` on any forge is open, and its head equals the head of a closed or merged cached PR `<n>`
- **THEN** the checkout is pinned to a temporary branch, removed, and the pin deleted only while it still points at that head, so a commit landing during removal stays on the pin
- **AND** if any cached PR `<n>` is open, or no closed PR `<n>` has that head, the worktree is kept
- **AND** without `--force`, a checkout another session occupies is kept under the same lease and handoff rules as a finished branch worktree

#### Scenario: Dry run and pickup hint

- **WHEN** `--dry-run` is passed
- **THEN** the same lines are printed with `would remove` in place of `removed`, and no worktree, branch, or remote ref changes
- **AND WHEN** pickup renders a worktree the same check would remove for the current agent, its line ends with `, prune candidate)`

#### Scenario: A finished worktree another session occupies is kept

- **WHEN** a finished, clean, owned branch worktree's own `.ateam/writer.json` names a session other than the caller's whose pid is alive or whose `last_seen` is within the writer-lease TTL, or its own `.ateam/state.json` holds a handoff for its branch from a session other than the caller's written less than 30 minutes ago
- **THEN** it is kept as `session <id> holds the lease (<age> ago)` or `session <id> handed off <age> ago; prune after 30m or pass --force`, and pickup does not mark it a prune candidate
- **AND** a lease or handoff from the caller's own session does not keep it, while with no resolvable caller session every lease and handoff does
- **AND** a writer lease or `.ateam/state.json` that exists but cannot be read or parsed keeps it as `writer lease unreadable: <err>` or `handoff cache unreadable: <err>`, while a missing one keeps nothing
- **AND WHEN** `--force` is passed, the worktree is removed as if neither rule applied (with `--dry-run`, listed as `would remove`), and every other keep reason still holds

#### Scenario: Pickup names the owner of a host-created branch

- **WHEN** pickup renders a linked worktree whose branch has no agent prefix (a host such as T3 Code created it on `t3/<topic>`)
- **THEN** its line names the agent from that worktree's handoff slot for the branch, else from its writer lease, else `owner unknown`


#### Scenario: The coordinator worktree is kept

- **WHEN** a clean linked worktree on the caller's `<agent>/coordinator` branch is labelled `merged (ancestor of main)` or `merged via #<n>`
- **THEN** without `--force` it is listed as `kept <agent>/coordinator: coordinator worktree; pass --force once the run has ended`, and pickup does not mark it a prune candidate
- **AND WHEN** `--force` is passed, it is removed under the same checks as any finished worktree (with `--dry-run`, listed as `would remove`)
