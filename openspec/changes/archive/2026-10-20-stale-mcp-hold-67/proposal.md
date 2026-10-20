# Proposal: Warn Owning Session About Stale MCP Server and Tidy Hold Display (`#67`)

## Why

During live verification of PR `#65`, two issues surfaced:
1. A long-running `ateam mcp` server started before `.ateam/mcp-processes.json` existed was only discovered via `ateam doctor`'s `ps` scan (`build unknown`) and was not surfaced to the owning session by `ateam hook`.
2. `ParseTasksMarkdown` and `BuildPickupSummary` prepended `words[0]` of the task line (`t.number`) onto captured `Hold:` lines and truncated at 104 bytes mid-word without an ellipsis, producing lines such as `Run Hold: ... is gr` and `Hold: Hold: ... or adapte`.

## What Changes

- Reuse `internal/bootstrap`'s MCP process enumerator (`FilterClientMCPProcesses`, `CheckClientMCPProcesses`, `InspectClientMCPProcesses`) in `internal/hook`, scoped to the invoking client's process tree (`pid`/`ppid` hierarchy, including shell wrappers) and keyed once per session (`mcp-checked-sessions.json` in `.ateam/`, independent of the single writer lease so interleaved sessions and `ClearWriterLease` handoffs preserve checked state), emitting:
  `MCP server PID n (build x) is older than HEAD; restart the client`
- Measured `ps -eo pid=,ppid=,lstart=,args=` inside the hook at `~32,600 us` (`> 5000 us` per-prompt budget); run the once-per-session scan on `session-start` (with fallback on the first `before-prompt` only if a session had no prior check for its `session_id`) so warm `before-prompt` and `session-start` hook latency remain `< 5000 us`.
- Tidy `Hold:` text in `ParseTasksMarkdown` (`internal/state/openspec.go`) and `BuildPickupSummary` (`internal/state/state.go:1853`) so hold text is printed once with a single `Hold:` prefix derived from the captured text and truncated on the preceding word boundary (regardless of token length, with a rune-safe fallback when no space exists) at 104 bytes with `…`.
