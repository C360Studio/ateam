# Change Proposal: `state-prune-229`

Approved-by:

## Why

Dogfood pass 10 (main 4debfe3, 2026-10-27) found three finished worktrees in one pickup: `gemini/daily-use-defaults` (`merged via #220`), `gemini/daily-use-guide` (`ahead of merged #208`), and a detached `codex-review-pr-213` review checkout. Each was correctly labelled, and none had been removed. Passes 6 and 8 showed the same thing. The Close rule (`.agents/protocol.md:44`) tells the claiming agent to remove its worktree and delete its remote branch once the merge lands. An agent whose session ends at auto-merge never gets a later turn to do that. The labels already exist in `worktreePRStatusAndAge` (`internal/state/state.go:1985-2042`), but nothing in `ateam` acts on them (#229).

## What Changes

- A new CLI command, `ateam state prune [--dry-run] [--agent <name>]`. It first refreshes state the way pickup does. It then removes every linked worktree the caller owns that is clean and finished: labelled `merged via #n` or `merged (ancestor of main)`, or a detached `<agent>-review-pr-<n>` checkout whose PR is closed. For each one it removes the worktree, deletes the local branch, and deletes the remote branch, but only when the remote branch still points at the local head. It prints one line per worktree saying whether it was removed or kept, and why. `--dry-run` prints the same lines and changes nothing.
- It never removes a worktree that is dirty, has an unknown dirty state, is labelled `ahead of merged` / `open PR` / `no PR` / `no commits yet`, belongs to another agent's prefix, is the current checkout, or is on the default branch. Those are listed with the reason they were kept.
- Pickup's worktree line ends with `, prune candidate` when the same check says the worktree can be removed. The check is pure and uses no I/O, so the hook budget is unchanged.
- The Close rule in `.agents/protocol.md`, `AGENTS.md` § Close, `ateam state --help`, and `README.md` name the command. The harness spec gains one Requirement.

## Track

`design`. The change adds a spec Requirement and a CLI command that adopting repositories can reach. See `design.md`.

## Non-goals

- No MCP action and no hook. Prune runs only when an agent or the owner asks for it.
- No removal of other agents' worktrees, and no `--all` flag (design Open Question 2).
- No change to how worktree labels are computed, or to the forge sync cap of 50 closed items.
- No cleanup of handoff slots. Pickup already prunes stale ones.

## Definition of Done

- `task verify` passes on a clean tree, `ateam doctor` parity passes, and `TestHookLatencyBudget` passes.
- A table test builds a temporary repository with linked worktrees. A squash-merged clean worktree is removed along with its local and remote branches. A dirty worktree is kept. A worktree under another agent's prefix is kept. A detached review checkout of a merged PR is removed. An `ahead of merged` worktree is kept. One row is a Gitea-forge fixture.
- `task mutate:check` reports `detection` when the dirty check is dropped, when the ownership check is dropped, and when the remote-head comparison is dropped.
