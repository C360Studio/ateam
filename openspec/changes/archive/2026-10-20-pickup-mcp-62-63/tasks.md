# Tasks: `pickup-mcp-62-63`

## 1. Implementation & Verification

- [x] 1.1 Reconcile `Worktree.MergedIntoMain` in Go from merged PR `head_sha` and emit a `Worktrees:` section (`branch`, `dirty`, `ahead`, `merged via #n` / `open PR #n`) in `BuildPickupSummary` (#62).
- [x] 1.2 Capture `state.BuildRevision()` at `ateam mcp` startup, report `build=<sha> mcp=<sha>` and `WARNING: MCP server binary is stale; restart the client to pick up <sha>` in MCP `pickup`, and list running `ateam mcp` processes and their build commits in `ateam doctor` (#63).
- [x] 1.3 Add unit tests with `// spec:` citations and run `task mutate:check`, `./bin/ateam doctor --check`, and `task verify`.
