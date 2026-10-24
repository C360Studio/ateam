# Change Proposal: `forge-sync-cap-149`

## Why

Forge sync asks for the 50 most recent issues and the 50 most recent pull requests with `state=all` (`internal/state/forge.go`, GitHub and Gitea). Once a repository has more than 50 of either, older open work drops out of the cache. Pickup then labels its worktree `no PR` and the board has no card for it (#149: PR 29 was open, but the cache's lowest number was 34).

## What Changes

- Open issues and pull requests are listed without the cap: GitHub `--state open --limit 1000`; Gitea `state=open`, page by page until an empty page or a page that adds no new numbers (the server may clamp the page size).
- Closed issues and pull requests keep a cap of 50, most recently updated first, so an old item that just closed or merged stays inside it (GitHub `--state closed --limit 50 --search sort:updated-desc`; Gitea pulls `state=closed&limit=50&sort=recentupdate`; the Gitea issues endpoint has no sort parameter, so closed issues stay newest-created first). The two lists are merged by number and the closed copy wins; the card shape does not change.
- The list queries for a forge run concurrently, so a sync costs its slowest call rather than the sum, inside the 3s pickup budget; closed GitHub issues skip comment expansion.
- The pickup summary's `synced` line records the closed-item cap, so a missing old closed card can be explained.

## Non-goals

- Paging closed items past the cap. The board does not need them. The `merged via #n` label reads the cached closed PR card (`internal/state/state.go` `worktreePRStatusAndAge`), which is why the closed list is ordered by last update: a just-merged PR is the most recently updated.
- Any change to card mapping, reconciliation, or comment expansion.
