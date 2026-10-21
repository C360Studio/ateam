# Change Proposal: `session-start-stale-cache-73`

## Why

`session-start` does not sync from GitHub/Gitea (to stay within the `<5ms` hook budget), so worktree PR labels in `BuildPickupSummary` read from the cached forge cards in `.ateam/state.json`. When `.ateam/state.json` was last synced hours earlier, labels like `open PR #64` or `unmerged` read as present-tense fact even after the PRs had merged, and review worktrees with no PR at all were labeled `unmerged` instead of `no PR` (#73).

## What Changes

- Label worktrees with no PR card on `st.Cards` (and not `main`/`master`/`detached` or `merged (ancestor of main)`) as `no PR` instead of `unmerged` in `(*State).WorktreePRStatus`.
- When the active forge cache timestamp (`SyncedAtFor(st.Forge)`) is at least `10m` old (`ForgeCacheStaleThreshold`), append `(cache <age>)` to forge-derived `WorktreePRStatus` labels (`open PR #64 (cache 4h)`, `no PR (cache 4h)`, `merged via #57 (cache 4h)`, `ahead of merged #55 (cache 4h)`, `closed without merge (#56) (cache 4h)`) and print `labels from forge cache synced <age> ago; run ateam state pickup --sync` under `Worktrees:`, while leaving live git-derived fields (`head`, `dirty`, `ahead`, `default branch`, `detached`, `merged (ancestor of main)`) uncaveated.
