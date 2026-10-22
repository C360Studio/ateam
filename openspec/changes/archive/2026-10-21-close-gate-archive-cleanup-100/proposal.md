# Change Proposal: `close-gate-archive-cleanup-100`

## Why

The Close gate didn't say *when* the OpenSpec archive move happens. Merged-but-unarchived changes showed in the queue on three of six dogfood passes. No instruction said who removes a finished worktree, and on `main` at `5b59d9d`, 9 of 14 worktrees were finished (#100).

## What Changes

- `AGENTS.md` Close bullet and `.agents/protocol.md` step 5 (with its bootstrap template) use the exact text from #100. The archive move is a branch commit before the PR is marked ready. After the merge is green on `main`, the claiming agent removes its worktree and remote branch, and a review checkout is removed when its record is posted.
- One new `review` row in section 3 of `AGENTS.md`, `.agents/rules-index.md`, and both bootstrap templates, so the row counts stay equal.
- Deferred to a follow-up: the optional checks in #100 (`finished_worktrees=` on session start, a `doctor` warning past 24h, `task worktree:prune`).
