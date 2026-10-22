# Change Proposal: `close-on-green-118`

## Why

The Close step waited for CI on `main` after the squash merge before the worktree and remote branch could go. The `main` ruleset already requires an up-to-date branch, so the PR run tests the exact tree that lands and the `main` run is a second opinion on it. On #115 that wait was 8 of 33 minutes, and the merge itself needed polling because auto-merge was off (#118).

## What Changes

- Close step (`AGENTS.md` bullet and section 3 `review` row, `.agents/rules-index.md`, `.agents/protocol.md` step 5): mark ready and arm auto-merge; once the required check is green on an up-to-date branch and the merge has landed, remove the worktree and remote branch. The `main` run is informational. Arming auto-merge past a previously red required job still needs the cause fixed or an owner waiver.
- Auto-merge enabled on the repository (owner, 2026-10-22).
- Not changed: the ruleset, merge-check, the archive-before-ready rule. The copies in `internal/bootstrap/templates.go` follow in #117 or a follow-up, since #117 edits the adjacent rows.
