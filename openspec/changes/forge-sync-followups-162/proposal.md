# Change Proposal: `forge-sync-followups-162`

Approved-by: Coby Leuschke (2026-10-27)

## Why

The #161 review (#149) and the #163 review left follow-ups on forge sync that did not block a merge (#162):

- `syncGiteaBoard` in `internal/state/forge.go` fetches comment bodies one issue at a time, each under its own 1.5s timeout. With many open issues that have comments, the loop can use up the 3s pickup budget. The sync still succeeds, but some comment bodies are missing.
- The comment above the Gitea open listing says an item updated mid-paging "cannot jump to a page already read". That is true. It leaves out that an item **closing** mid-paging shifts later items, so one can be missed for that sync.
- `syncGitHubBoard` gives the open-issues-with-comments call `min(750ms, deadline/3)` under a parent deadline. On this repository the call takes about 0.8s, so under the 3s pickup budget it usually falls back and drops comment bodies.
- Closed lists come from `gh --search`, so a just-merged PR can label its worktree `no PR` until GitHub's index catches up. The harness spec states this; the code does not.
- `bootstrap.CheckBuildProvenance` is a one-line wrapper over `BuildProvenanceStatus`, and only `internal/bootstrap/parity_test.go` calls it.

The timing-based budget test the issue names is already gone: `TestGitHubSyncRunsListsConcurrently` uses a rendezvous fake (#188). No change there.

## What Changes

- Gitea comment fetches run concurrently with at most 8 in flight, under the parent context. A failed fetch still leaves that issue without comment bodies, as today.
- Under a parent deadline, the GitHub comments call gets `deadline/2` (no fixed 750ms cap), and the comment-free fallback gets the rest. With no deadline, `ghCommentTimeout` stays 2.5s.
- The Gitea paging comment names the close-mid-paging miss, and a code comment at the GitHub closed lists names the search-index lag.
- `CheckBuildProvenance` is deleted. Its tests call `BuildProvenanceStatus`.

## Track

`quick`. One package's sync internals plus one unexported-in-practice helper in `internal/bootstrap`. No spec Requirement is added or changed, no adopter-reachable surface, boot path, durability/ack behavior, or gate file.

## Non-goals

- Paging GitHub open lists past 1000 (the existing `// shortcut:` stays).
- Removing the close-mid-paging miss itself, which needs a second pass or ID-based paging.
- Changing fallback semantics, token roles, or the 3s pickup budget.

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor` parity green.
- A Gitea fake test proves comment fetches overlap and never exceed 8 in flight, and that each issue's bodies land on its own card. `task mutate:check` reports `detection` for a sequential loop and for a removed bound.
- A test proves the GitHub comments call receives about half of a parent deadline, and `task mutate:check` reports `detection` when it is reverted to the 750ms cap.
- `go test -race ./internal/state/... ./internal/bootstrap/...` passes.
