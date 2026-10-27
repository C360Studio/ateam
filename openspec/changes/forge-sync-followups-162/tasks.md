## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#162). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 Gitea comment fetches in `syncGiteaBoard` run concurrently, at most 8 in flight, under the parent context, with a fake-server test for overlap, the bound, and per-issue bodies, proven able to fail by `task mutate:check` (#162). Evidence: `giteaCommentFetchLimit` and the `slots` semaphore in `fetchGiteaRESTCards` (`internal/state/forge.go`); `TestGiteaSyncFetchesCommentsConcurrentlyWithinLimit` in `internal/state/forge_test.go`. `task mutate:check -runs 3` reported `detection` for a sequential loop (peak 1) and for a removed bound (peak 9); `forge.go` SHA-256 `3a2bf9f3...492a9cd` before and after.
- [x] 2.2 The GitHub comments call takes `deadline/2` under a parent deadline, capped at `ghCommentTimeout`, with tests proven able to fail by `task mutate:check`; the stale 750ms test comments are updated (#162). Evidence: `ghCommentBudget` in `internal/state/forge.go`. `TestGitHubCommentBudgetIsHalfTheParentDeadline` checks the arithmetic: `detection` for the old `min(750ms, deadline/3)` in the helper and for a removed `ghCommentTimeout` cap. `TestGitHubSyncGivesSlowCommentsTheDeadlineBudget` drives `SyncForgeBoard` with a 1s comments call: `detection` (3 runs) for a call site reverted to `750 * time.Millisecond`. The cap came from the internal Go review: uncapped, the UI's 10s sync waited 5s on a hung call and probe 4c took 15s. `forge.go` SHA-256 `3a2bf9f3...492a9cd` before and after; comments in `forge_test.go` and `forge_settings_test.go` updated.
- [x] 2.3 The Gitea paging comment names the close-mid-paging miss, and the GitHub closed-list code names the search-index lag (#162). Evidence: comments above the Gitea `opens` listing and at `closedSort` in `internal/state/forge.go`.
- [x] 2.4 `bootstrap.CheckBuildProvenance` is deleted and `internal/bootstrap/parity_test.go` calls `BuildProvenanceStatus` (#162). Evidence: no remaining Go reference; `go test -race ./internal/bootstrap/` passes.

## 3. Verify

- [ ] 3.1 `task verify` green on a clean tree and `ateam doctor` parity green (#162).
