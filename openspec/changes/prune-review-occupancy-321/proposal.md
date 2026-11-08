# Change Proposal: `prune-review-occupancy-321`

Approved-by: Coby Leuschke (2026-11-08)

## Why

`PruneVerdict` (`internal/state/prune.go:77`) returns `review of closed PR #<n>` for a clean, owned, detached review checkout whose head is a closed PR's head, before it runs the occupancy check that merged branch worktrees get (`prune.go:79-84`). Every Claude session shares the `claude-review-pr-*` prefix, so one session's `ateam state prune` removes a review checkout another session is still standing in, if that session is idle and the tree is clean (#321, HAAG R2, finding F10).

## What Changes

- Without `--force`, a detached review checkout of a closed PR goes through `occupiedBy` before removal, exactly as a merged branch worktree does: another session's live or recent writer lease, or its handoff under 30 minutes old, in that checkout keeps it.
- The `A detached review checkout of a closed PR is removed` scenario of `Prune finished worktrees the caller owns` in `openspec/specs/harness/spec.md` gains the occupancy clause.

## Track

`quick`: one scenario clause on an existing Requirement in one capability, no new exported surface, no gate file.

## Non-goals

- Where it stops: a reviewer started with `ATEAM_ROLE=review` (herdr, terminal) writes no writer lease (#315), and a reviewer rarely hands off, so occupancy cannot see it; this fix protects only sessions that left a lease or handoff in the review checkout (a T3 Codex reviewer does). Named in the PR, not fixed here.
- No change to the dirty, head-match, open-PR, or ownership rules, or to `--force`.

## Definition of Done

- A `PruneVerdict` test keeps a closed PR's review checkout under another session's live lease and fresh handoff, removes it under the caller's own lease and under `--force`; `task mutate:check` reports `detection` with the new check removed.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
