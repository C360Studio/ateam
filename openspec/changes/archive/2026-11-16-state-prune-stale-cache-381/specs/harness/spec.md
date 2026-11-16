# harness Specification Delta

## MODIFIED Requirements

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
- **THEN** when a linked worktree's own `.ateam/state.json` holds a newer forge sync timestamp for a provider than the current state's `SyncedAtFor`, its PR cards for that worktree's branch are consulted with that newer timestamp
- **AND** a worktree with no PR card on the board or in a newer linked-worktree cache (and not `main`/`master`/`detached` or an ancestor of `main` with `ahead == 0`) is labeled `PR unknown (cache predates branch)` when the supplying forge cache timestamp is non-zero and predates the worktree's `HEAD` commit timestamp (`head_committed_at`), and otherwise `no PR` rather than `unmerged`
- **AND** when the supplying forge cache timestamp (`SyncedAtFor` for the matched card's provider, or the active forge for `no PR` / `PR unknown (cache predates branch)`) is at least `10m` old, PR/forge-derived worktree labels append `(cache <age>)` (e.g. `open PR #64 (cache 4h)`, `no PR (cache 4h)`, `PR unknown (cache predates branch) (cache 4h)`, `merged via #57 (cache 4h)`), `Worktrees:` prints `labels from forge cache synced <age> ago; run ateam state pickup`, and live git-derived fields (`head`, `dirty`, `ahead`, and `default branch` / `detached` / `merged (ancestor of main)` / `no commits yet`) carry no cache caveat
- **AND** a detached checkout is labeled `detached` on every surface, including after a live head read that names it `HEAD`, and a branch with no PR card whose head equals the `main` head is labeled `no commits yet` rather than `merged (ancestor of main)`

#### Scenario: Open work is never capped

- **WHEN** a forge holds more than 50 open issues or more than 50 open pull requests
- **THEN** synchronization SHALL list every open issue and pull request (GitHub `--state open` with a limit of 1000; Gitea `state=open`, page by page until an empty page or a page that adds no new numbers) and at most the 50 most recently updated closed ones (Gitea closed issues: the 50 newest, since the issues endpoint cannot sort), merged by number with the closed copy winning, so a worktree on an old open pull request's branch is labelled `open PR #<n>` and never `no PR`, and one on a just-merged old pull request is labelled `merged via #<n>` once GitHub's search index holds the merge (until then a sync may briefly label it `no PR`)
- **AND** the list queries for one forge SHALL run concurrently, so the sync fits the 10s per-forge best-effort sync budget
- **AND** the pickup summary's `synced` line names the closed-item cap

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
