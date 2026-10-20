# Change Proposal: `pickup-mcp-62-63`

## Why

Round-3 dogfooding on `main` at `e37563b` identified two agent-seat visibility gaps:
1. **Worktree merge status in Go snapshot & `pickup` (#62):** `SnapshotGit` only set `Worktree.MergedIntoMain` to `ahead == 0`, which is never true for squash-merged PRs, and `BuildPickupSummary` omitted worktrees completely so CLI/MCP `pickup` could not distinguish merged/abandoned worktrees from active/open-PR worktrees.
2. **Stale running `ateam mcp` binary detection (#63):** A long-running `ateam mcp` server continued running an older binary after `task build` without surfacing `mcp=<sha>` or warning when its build commit differed from `./bin/ateam` on disk or checkout `HEAD`.

## What Changes

1. **Go worktree merge reconciliation and `pickup` `Worktrees:` section (#62):**
   - Add `(*State).ReconcileWorktrees()` and `(*State).WorktreePRStatus(wt Worktree)` in `internal/state/state.go`, matching the UI rule (`done` PR card for the branch with `merged: true` and `head_sha` matching `wt.Head` sets `MergedIntoMain = true`, keeping `ahead == 0` as fallback and `false` when new commits exist ahead of a merged PR or when an open PR exists).
   - Call `ReconcileWorktrees()` during `Refresh`, `MergeForgeCards`, and `BuildPickupSummary`, and print a `Worktrees:` section in `BuildPickupSummary` showing branch, dirty, ahead count, and `merged via #n` / `open PR #n`.
2. **MCP server build provenance & staleness warning (#63):**
   - Capture `state.BuildRevision()` at `ateam mcp` startup, record active MCP server instances (`pid`, `build_commit`) in `.ateam/mcp-processes.json`, and include `build=<disk-or-current-sha> mcp=<server-sha>` in MCP `pickup` responses (`ateam_state` `pickup` and `ateam://state/pickup`).
   - Emit `WARNING: MCP server binary is stale; restart the client to pick up <sha>` in MCP `pickup` when the server's build commit differs from `./bin/ateam` on disk or checkout `HEAD`, and list running `ateam mcp` processes alongside `ateam serve` in `ateam doctor`.
