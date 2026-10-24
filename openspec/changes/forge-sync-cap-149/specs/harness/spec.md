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
- **THEN** a worktree with no PR card on the board (and not `main`/`master`/`detached` or an ancestor of `main` with `ahead == 0`) is labeled `no PR` rather than `unmerged`
- **AND** when the supplying forge cache timestamp (`SyncedAtFor` for the matched card's provider, or the active forge for `no PR`) is at least `10m` old, PR/forge-derived worktree labels append `(cache <age>)` (e.g. `open PR #64 (cache 4h)`, `no PR (cache 4h)`, `merged via #57 (cache 4h)`), `Worktrees:` prints `labels from forge cache synced <age> ago; run ateam state pickup`, and live git-derived fields (`head`, `dirty`, `ahead`, and `default branch` / `detached` / `merged (ancestor of main)` / `no commits yet`) carry no cache caveat
- **AND** a detached checkout is labeled `detached` on every surface, including after a live head read that names it `HEAD`, and a branch with no PR card whose head equals the `main` head is labeled `no commits yet` rather than `merged (ancestor of main)`

#### Scenario: Open work is never capped

- **WHEN** a forge holds more than 50 open issues or more than 50 open pull requests
- **THEN** synchronization SHALL list every open issue and pull request (GitHub `--state open` with a limit of 1000; Gitea `state=open`, page by page until an empty page or a page that adds no new numbers) and at most the 50 most recently updated closed ones (Gitea closed issues: the 50 newest, since the issues endpoint cannot sort), merged by number with the closed copy winning, so a worktree on an old open pull request's branch is labelled `open PR #<n>` and never `no PR`, and one on a just-merged old pull request is labelled `merged via #<n>`
- **AND** the list queries for one forge SHALL run concurrently, so the sync fits the 3s per-forge pickup budget
- **AND** the pickup summary's `synced` line names the closed-item cap
