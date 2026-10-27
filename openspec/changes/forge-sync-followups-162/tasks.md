## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#162). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 Gitea comment fetches in `syncGiteaBoard` run concurrently, at most 8 in flight, under the parent context, with a fake-server test for overlap, the bound, and per-issue bodies, proven able to fail by `task mutate:check` (#162). Evidence: `giteaCommentFetchLimit` and the `slots` semaphore in `fetchGiteaRESTCards` (`internal/state/forge.go`); `TestGiteaSyncFetchesCommentsConcurrentlyWithinLimit` in `internal/state/forge_test.go`. `task mutate:check -runs 3` reported `detection` for a sequential loop (peak 1) and for a removed bound (peak 9); `forge.go` SHA-256 `e75f6731...390a402` before and after.
- [x] 2.2 The GitHub comments call takes `deadline/2` under a parent deadline, with a test proven able to fail by `task mutate:check`; the stale 750ms test comments are updated (#162). Evidence: `ghCommentBudget` in `internal/state/forge.go`; `TestGitHubCommentBudgetIsHalfTheParentDeadline`. `task mutate:check` with the old `min(750ms, deadline/3)` reported `detection` (750ms); comments in `forge_test.go` and `forge_settings_test.go` updated.
- [x] 2.3 The Gitea paging comment names the close-mid-paging miss, and the GitHub closed-list code names the search-index lag (#162). Evidence: comments above the Gitea `opens` listing and at `closedSort` in `internal/state/forge.go`.
- [x] 2.4 `bootstrap.CheckBuildProvenance` is deleted and `internal/bootstrap/parity_test.go` calls `BuildProvenanceStatus` (#162). Evidence: no remaining Go reference; `go test -race ./internal/bootstrap/` passes.

## 3. Verify

- [ ] 3.1 `task verify` green on a clean tree and `ateam doctor` parity green (#162).
