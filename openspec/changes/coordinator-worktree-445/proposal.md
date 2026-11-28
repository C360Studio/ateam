# Proposal: coordinator-worktree-445
Approved-by:

## Track

`quick`: it modifies two existing harness Requirements and adds none, touches one capability, adds no exported surface, and is no boot path, durability or gate file. The code is `internal/state/forge.go` (the GitHub handoff post) and `internal/state/prune.go` (prune's keep rule).

## Decisions you are making by approving

- **A handoff on a branch with no open pull request posts nothing and prints nothing on GitHub, as it already does on Gitea.** The coordinator's handoff no longer prints `gh pr comment claude/coordinator: exit status 1`, and the coordinate skill needs no `--no-post`. A forge read that fails still prints `forge PR comment skipped: <err>`.
- **A handoff on a branch whose pull request has already merged or closed no longer comments on that pull request on GitHub.** Today GitHub finds the merged PR by branch name and posts there; Gitea and the protocol already say "the branch's open PR". A developer who hands off after their merge keeps the handoff in the local cache only.
- **A worktree on `<agent>/coordinator` is never removed by `ateam state prune` and never marked a prune candidate in pickup, unless `--force` is passed.** It is listed as `kept claude/coordinator: coordinator worktree; pass --force once the run has ended`. The coordinator waits as long as it likes on a developer or on you without another session removing its checkout. Its pickup label stays the git fact (`merged (ancestor of main)` or `no commits yet`); only the prune hint goes.

## Why

#445, from the v0.4.7 coordinator dogfood (notes 2 and 39). The coordinator contract (`.agents/contracts/coordinator.md:3`) gives the coordinator a worktree with no PR and no commits of its own. So `ateam state handoff` fails on GitHub's `gh pr comment <branch>` (`internal/state/forge.go:1046`) where the Gitea path already returns quietly (`:1103`), and once `main` moves past its head, pickup labels it `merged (ancestor of main), prune candidate` (`internal/state/prune.go:84`, `state.go:3354`); prune spares it only for 30 minutes after a handoff.

## What Changes

1. `PostPRHandoffComment` on GitHub first lists the branch's open PR (`gh pr list --head <branch> --state open`); none returns an empty result with no error, and the comment read, send and head read address that PR's number.
2. `PruneVerdict` keeps a branch named `<agent>/coordinator` unless `force`; pickup's hint follows, since it calls the same check.
3. Harness spec: one scenario each under the two Requirements below.

## Scope & Non-Goals

- **In-bounds:** `internal/state/forge.go`, `internal/state/prune.go`, their tests (`handoff_post_test.go`, `commitstatus_test.go`, `token_test.go`, `state_test.go`, `prune_test.go`), the harness spec delta.
- **Non-goals:** the coordinate skill and its template (no `--no-post` needed); the worktree's pickup label and the board's `merged` column; removing the coordinator worktree at the end of a run; parsing the coordinator's handoff for units in flight; Gitea, which already skips; the MCP handoff, which already reports `PR comment: skipped (no open PR for branch)`.

## Definition of Done

- Tests: GitHub no open PR posts nothing and errs nothing; a failed list errs and sends nothing; a merged-only branch posts nothing; `claude/coordinator` labelled `merged (ancestor of main)` is kept, removed with `--force`, and pickup shows no prune hint; each shown able to fail with `task mutate:check`.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
