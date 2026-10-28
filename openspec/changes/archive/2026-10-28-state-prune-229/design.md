# Design: `state-prune-229`

Drafted by claude (opus) in the design skill. ADR 0003 sets this role's default at frontier tier and high effort, so the owner may want a second read.

## Context

- `internal/state/state.go:1985-2042` `worktreePRStatusAndAge` computes each label. Detached and default-branch checkouts return before any card lookup. `open PR #n` wins over every done card. `ahead of merged #n` means the local head differs from the merged PR head and there are commits ahead. `merged via #n` means the head matches the PR head, or the PR has no head and the branch has no commits ahead. `merged (ancestor of main)` is derived from git alone. `no commits yet` is a branch sitting at main's head.
- `state.go:1883-1927` `ReconcileWorktrees` sets `MergedIntoMain` from the same done cards.
- `state.go:2407-2430` `Refresh` sets `Dirty` per worktree from `git status --porcelain`. `Dirty` is nil when the path is missing (`state.go:495`). `AheadCount` is counted against `main`.
- `state.go:2490-2497` `inferAgentFromBranch` checks the `claude|codex|gemini|agy` + `/` prefix. `parseWorktrees` (`state.go:2458`) sets `Agent` from it.
- `state.go:2840-2875` `writePickupWorktrees` prints `  - <branch> (head=…, dirty=…, ahead=…, <label>)`. It runs inside session-start (`internal/hook/hook.go` → `BuildPickupSummary`), so it must stay free of I/O.
- `state.go:1847` `selectDoneBranchPR` and `cardPRNumber` find the done card for a branch. Cards are capped at the 50 most recent closed items (`openspec/specs/harness/spec.md:126`).
- `cmd/ateam/main.go:234-251` `runStatePickup`: `RefreshWithBestEffortSync(ctx, root, 30s)` (`state.go:2232`), then `ResolveActiveAgentForSession`. The `runState` dispatch is at `main.go:134-160` and `stateUsage` at `main.go:166`.
- `.agents/protocol.md:44` Close: the agent removes its worktree and remote branch by hand. Review checkouts are named `<agent>/review-pr-<n>` there, while `.agents/skills/review-pr/SKILL.md:22` creates them at `.worktrees/<agent>-review-pr-<n>` (`git worktree add --detach`). The dogfood checkout was `codex-review-pr-213-d850` (#229).
- Nothing in the repository runs `git worktree remove` or `git push --delete` today (`grep -rn -e '"worktree", "remove"' -e '"--delete"' internal cmd` matches nothing outside tests).

## Goals / Non-Goals

Goals: one command that does the Close cleanup safely and can be run later, plus a pickup hint that points to it. Non-goals are listed in `proposal.md`.

## Decisions

### D1. What counts as finished

- **A (recommended): reuse the label.** Remove only `merged via #n` and `merged (ancestor of main)`, plus a detached checkout named `<agent>-review-pr-<n>[-suffix]` whose card for PR n is done. Everything else is kept with its label as the reason. Cost: one pure function over existing fields. Blast radius: `internal/state` only. It rules out pruning `closed without merge` branches, which may hold abandoned work someone wants back.
- **B: recompute from git** (`git branch --merged`, `git cherry`). This gives the same answer for merge commits but misses squash merges, which are the common case here. It would also duplicate the label logic.
- **Adopter seam:** an adopter never calls this logic directly; they see `removed` / `kept` lines. If they do nothing, finished worktrees keep piling up as they do now.

### D2. How fresh the labels must be

- **A (recommended): refresh like pickup first** (`RefreshWithBestEffortSync`, 30s), and print the `labels from forge cache synced <age> ago` caveat when the sync failed. Forge-derived labels can only lead to a removal that is also safe on git evidence: the head equals the merged PR head, which is pushed by definition, and the tree is clean. So a stale cache can at worst keep something that is finished. It cannot remove something that is not.
- **B: refuse every forge-derived removal unless synced in the last 10 minutes.** This is stricter but fails offline. The safety argument in A already covers the stale case.
- **Adopter seam:** prune needs the same forge access pickup already uses (`gh` or `tea`/API). Offline, only `merged (ancestor of main)` worktrees are removed.

### D3. Ownership

- **A (recommended): owner = branch prefix (or review-path prefix) equal to the resolved caller agent**, with `--agent` overriding it the same way it does on pickup. Unprefixed branches and other agents' branches are kept as `owned by <agent>` or `no agent prefix`. This enforces the protocol rule "never remove a worktree you do not own" mechanically.
- **B: also accept an `--all` flag** for the owner, who owns everything. This is convenient for the backlog seen in pass 10, but it is a second removal path to test and goes beyond what was asked. Deferred to Open Question 2.
- **Adopter seam:** a repository whose agents do not use `<agent>/` prefixes gets `kept: no agent prefix` on every line. The fix is to follow the Claim gate's branch naming.

### D4. Removal mechanics

- **A (recommended): plain git.** Run `git worktree remove <path>` without `--force`, so git itself refuses a worktree that is dirty or has untracked files at removal time, which closes the race with a concurrent edit. Then `git branch -D <branch>`; `-D` is needed because a squash merge is not an ancestor. Then `git push origin --delete <branch>` only if `git ls-remote origin refs/heads/<branch>` equals the local head; a remote branch that has moved is kept and reported. This works the same on GitHub and Gitea.
- **B: forge API ref deletion** (`gh api -X DELETE …/git/refs/heads/<b>`, Gitea `DELETE /branches/<b>`). It would also cover a forge's own auto-delete, but it needs two code paths and tokens for something git already does.
- Each step's failure is printed and stops that worktree. Other worktrees continue, and the exit code is 1 if any step failed. Prune does not take the writer lease: it writes no handoff slot, and git serializes worktree operations.

### D5. Surface

- **A (recommended): CLI only**, plus the `, prune candidate` pickup suffix and the Close-rule text. This leaves the MCP schema, its 300-token budget, and hook behaviour untouched.
- **B: an `ateam_state` `prune` action.** An MCP-only agent could then close its own work, but a destructive action behind a multiplexed tool is easier to call by mistake. The issue asks for neither.
- **Adopter seam:** the new subcommand appears in `ateam state --help` of an adopting repository after `ateam init`. The pickup suffix changes the worktree line, so an anchored `<label>)$` grep would see `, prune candidate)`.

## Risks / Trade-offs

- A worktree whose merged PR has dropped out of the 50-item closed cache reads `no PR` and is kept. This is safe and only means less gets cleaned up.
- A worktree's path can be the caller's own working directory. It is excluded by comparing the path with `root`.
- Some desktop hosts create and remove `.worktrees/<agent>-*` themselves. Pruning underneath them may leave the host showing a missing worktree (Open Question 1).

## Migration Plan

None. This adds a new command and output suffix, with no state schema change.

## Open Questions

1. **Ruled (owner, 2026-10-28): A.** When a host app manages a worktree's lifecycle, should prune still remove the worktree (A: yes; git is the truth and the host sees it gone), or should it add `--branches-only` to delete only the local and remote branches (B)? This changes task 3.1.
2. **Ruled (owner, 2026-10-28): deferred.** Is `--all` (D3 B) wanted now for the owner's backlog, or deferred? The recommendation is to defer.
3. Not blocking: the protocol's `<agent>/review-pr-<n>` disagrees with the review-pr skill's `<agent>-review-pr-<n>` directory. The proposal is to match the skill's directory name, allowing a `-<suffix>`, and to fix the protocol text in task 5.1.
