## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#162). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Gitea comment fetches in `syncGiteaBoard` run concurrently, at most 8 in flight, under the parent context, with a fake-server test for overlap, the bound, and per-issue bodies, proven able to fail by `task mutate:check` (#162).
- [ ] 2.2 The GitHub comments call takes `deadline/2` under a parent deadline, with a test proven able to fail by `task mutate:check`; the stale 750ms test comments are updated (#162).
- [ ] 2.3 The Gitea paging comment names the close-mid-paging miss, and the GitHub closed-list code names the search-index lag (#162).
- [ ] 2.4 `bootstrap.CheckBuildProvenance` is deleted and `internal/bootstrap/parity_test.go` calls `BuildProvenanceStatus` (#162).

## 3. Verify

- [ ] 3.1 `task verify` green on a clean tree and `ateam doctor` parity green (#162).
