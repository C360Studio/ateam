# Tasks: Warn Owning Session About Stale MCP Server and Tidy Hold Display (`#67`)

## 1. Implementation & Verification

- [x] 1.1 Extend `internal/bootstrap` MCP process enumeration with `PPID` parsing, client process-tree filtering (`FilterClientMCPProcesses`), and one-line session warning formatting (`CheckClientMCPProcesses`, `InspectClientMCPProcesses`).
- [x] 1.2 Wire once-per-session stale client MCP check in `internal/hook/hook.go` keyed on `sessionID` (`.ateam/mcp-checked-sessions.json`, independent of `.ateam/writer.json` so interleaved sessions and `ClearWriterLease` handoffs preserve checked state) and record the `ps` latency measurement (`~32,600 us` cold `ps` scan vs `~250 us` warm `before-prompt`).
- [x] 1.3 Fix `ParseTasksMarkdown` (`internal/state/openspec.go`) and `BuildPickupSummary` (`internal/state/state.go`) so `Hold:` items render a single `Hold:` prefix without re-prefixing leading task words/numbers and truncate on the preceding word boundary (regardless of token length) at 104 bytes with `…`.
- [x] 1.4 Add unit tests and run non-destructive mutation checks (`task mutate:check`) and `task verify`.
