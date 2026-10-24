# Change Proposal: `forge-sync-cap-149`

## Why

Forge sync asks for the 50 most recent issues and the 50 most recent pull requests with `state=all` (`internal/state/forge.go`, GitHub and Gitea). Once a repository has more than 50 of either, older open work drops out of the cache. Pickup then labels its worktree `no PR` and the board has no card for it (#149: PR 29 was open, but the cache's lowest number was 34).

## What Changes

- Open issues and pull requests are listed without the cap: GitHub `--state open --limit 1000`; Gitea `state=open`, page by page until a short page.
- Closed issues and pull requests keep a cap of 50 (`--state closed --limit 50`; Gitea `state=closed&limit=50`). The two lists are merged by number, and the card shape does not change.
- The pickup summary's `synced` line records the closed-item cap, so a missing old closed card can be explained.

## Non-goals

- Paging closed items past the cap. The board does not need them, and the `merged via #n` label reads the merge from git, not from the cache.
- Any change to card mapping, reconciliation, or comment expansion.
