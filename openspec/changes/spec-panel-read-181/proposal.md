# Change Proposal: `spec-panel-read-181`

Approved-by:

## Why

A proposal that waits for owner approval lives in an agent's worktree, on its branch. The board's OpenSpec tab only shows `openspec/changes/` in the selected checkout (`ScanOpenSpecQueue(root)`, `internal/state/openspec.go:306`). That checkout is usually `main`, so the changes the owner needs to read do not show. The tab also shows task counts and caveats, but not the proposal text, the `Approved-by:` line or the draft PR. #181 asks for a read view, which can land before its Approve write. The owner ruled on 2026-11-10 that the read view ships on its own, quick track, and the write follows as a separate design-track change.

## What Changes

- `/api/openspec` (`internal/web/server.go:650`) also returns the in-flight changes found in every worktree from `git worktree list`. Each one carries its `branch`, its `worktree` path, and the `proposal` markdown read from that worktree's change directory. A change that appears in several worktrees is listed once per branch. The existing `changes` field stays as it is.
- `OpenSpecChange` already carries `approved_by` and `approval` (#272). The per-worktree list adds only `branch`, `worktree` and `proposal` beside it.
- The OpenSpec tab (`ui/src/routes/+page.svelte:860`) gains a spec panel. For each change it shows the proposal, task progress, holds with their markers, the `Approved-by` value and approval state (approved, stale or missing), and a link to the branch's PR from the synced forge cards when there is one. Changes holding on `OPEN-Q` sort first.
- The proposal is shown as preformatted text. There is no markdown renderer in `ui/package.json`, and this change does not add one.

## Track

`quick` (owner, 2026-11-10). It modifies one existing Requirement (*Board state and authenticated API access*) by adding a scenario. It adds no Requirement, no exported surface reachable from an adopting repository, and no write.

## Non-goals

- No Approve button and no write of any kind. That is the separate design-track change for #181.
- No markdown rendering dependency, and no editing of proposals or tasks.
- No change to `ateam state spec-queue`, `spec:check`, the commit statuses, or the MCP tools.

## Definition of Done

- A server test with two worktrees, one holding a change on `OPEN-Q`, shows the change listed with its branch, its proposal text, and an approval state that matches the bytes on disk.
- A UI test checks that an `OPEN-Q` change sorts first and that the PR link appears only when a card for that branch exists.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before the PR is marked ready.
