# Tasks: state-prune-stale-cache-381

## 1. Approval

- [x] 1.1 Record owner approval for #381 (`Approved-by: Coby Leuschke (2026-11-16)`).

## 2. Implementation & Verification

- [x] 2.1 Raise best-effort forge sync timeout in `RefreshWithBestEffortSync`, record `Worktree.HeadCommittedAt` in `SnapshotGit`, consult linked worktrees' newer cached PR cards, and report `PR unknown (cache predates branch)` when the supplying forge cache predates the branch HEAD commit.
- [x] 2.2 Print the `WARNING: Forge board sync failed (serving cached board, synced=...)` line in `state.Prune` when `st.BoardSyncError` is non-empty.
- [x] 2.3 Add unit tests in `internal/state/prune_test.go` and `internal/state/state_test.go`, run `task mutate:check`, `task verify`, `ateam doctor --check`, and `ateam bench --selftest`, then archive this change.
