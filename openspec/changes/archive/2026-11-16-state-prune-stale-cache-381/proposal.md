# Proposal: state-prune-stale-cache-381

- **Track:** spec
- **Issue:** #381
- **Approved-by:** Coby Leuschke (2026-11-16) sha256:edb34dc97985

## Why

When `ateam state prune` or `ateam state pickup` runs from the main checkout after a PR is created and merged inside a linked worktree:

1. Best-effort forge sync in `RefreshWithBestEffortSync` currently enforces a 3s per-forge timeout while `syncGitHubBoard` spawns `gh` subcommands with 5s timeouts, frequently killing `gh issue list` or `gh pr list` on larger repositories (`signal: killed`) and falling back to an older cache.
2. `ateam state prune` never prints `BoardSyncError` when forge sync fails and falls back to the cached board, so `prune` silently prints `kept <branch>: no PR (cache 14m)` as if `no PR` were a known fact.
3. Even when a linked worktree's own `.ateam/state.json` has a newer forge sync holding the branch's PR card, `ReconcileWorktrees` and `WorktreePRStatus` in the main checkout only inspect the main checkout's `st.Cards`.
4. When no checkout's cache holds a PR card for a branch and the supplying forge cache timestamp predates the branch's `HEAD` commit timestamp (`git log -1 --format=%ct`), labeling the worktree `no PR` states as fact what is actually `PR unknown (cache predates branch)`.

## What Changes

1. **Raise best-effort forge sync timeout (`internal/state/state.go`):** Raise the per-forge timeout in `RefreshWithBestEffortSync` (`bestEffortSyncTimeout`) from `3s` to `10s` so the 5s `gh` list calls and comment budget can complete without being killed mid-flight.
2. **Surface forge sync fallback warning in `Prune` (`internal/state/prune.go`):** When `st.BoardSyncError` is non-empty, `Prune` prints the same `WARNING: Forge board sync failed (serving cached board, synced=<timestamp>): <err>` line before its worktree verdict lines.
3. **Consult linked worktrees' newer cached PR cards (`internal/state/state.go`):** When a linked worktree has its own `.ateam/state.json` whose forge sync timestamp for a card's provider is newer than the current state's `SyncedAtFor` for that provider, `WorktreePRStatus` / `ReconcileWorktrees` / `worktreeReviewSuffix` consult those newer PR cards for that worktree's branch (and compute any `(cache <age>)` caveat from that newer sync timestamp).
4. **Report `PR unknown (cache predates branch)` (`internal/state/state.go`):** Record each worktree's `HeadCommittedAt` (`git log -1 --format=%ct`) in `SnapshotGit`. When a branch worktree has no PR card in the current or linked-worktree cache and `!syncedAt.IsZero() && wt.HeadCommittedAt.After(syncedAt)`, report `PR unknown (cache predates branch)` (with `(cache <age>)` when `syncedAt` is older than `ForgeCacheStaleThreshold`) instead of `no PR`.

## Scope & Non-Goals

- **In scope:** `internal/state/state.go`, `internal/state/prune.go`, unit tests in `internal/state/prune_test.go` and `internal/state/state_test.go`, and `openspec/specs/harness/spec.md`.
- **Non-goals:** Making live per-branch network calls inside `WorktreePRStatus`, `PruneVerdict`, or the `<10ms` `session-start` hook path.
