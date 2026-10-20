# Change: Remediate Fable dogfooding state, lease, board cache, and binary provenance blockers (#39-#42)

implemented-by: gemini

Issue: https://github.com/C360Studio/ateam/issues/44
Addresses: #39, #40, #41, #42, #44
Owner: Gemini (`gemini/fable-blockers-39-42`)

## Why

Dogfooding `ateam` in `ateam` exposed four blocking defects:
1. **#39 (Stale handoff survives HEAD drift and `pre-compact`):** `BuildPickupSummary` noted HEAD drift (`02e913f -> ea0c784`) on a long-merged PR (#2) but still injected its 5-field goal block on every `session-start`, and `pre-compact` re-saved that stale handoff as a fresh checkpoint.
2. **#40 (Writer lease false positive via MCP and ephemeral hook PID):** `WriteWriterLease` recorded `os.Getpid()` of the `<5ms` hook process (which immediately exits) instead of the parent client process PID, and `ateam mcp` evaluated `BuildPickupSummary` with an empty `CurrentSessionID`, warning on every `pickup` that the invoking session was a foreign writer.
3. **#41 (Stale board cache and unpruned `ATEAM-*` demo cards):** `pickup` and `board` served hours-old `.ateam/state.json` snapshots without attempting a bounded forge sync or surfacing `board_synced_at`, and `MergeForgeCards` permanently retained unbacked `ATEAM-*` cards alongside live forge cards.
4. **#42 (Binary build provenance and multi-server contamination):** `ateam version` and `[ateam]` session-start output omitted build commit provenance, and `ateam doctor` neither checked whether the running binary's commit was an ancestor of checkout `HEAD` nor warned when multiple `ateam serve` processes were running concurrently.

## What Changes

- **#39 (`internal/state/state.go`, `internal/hook/hook.go`):** Evaluate handoff staleness (`EvaluateHandoffStaleness` / `IsHandoffStale`) when `CheckpointSHA` drifts from `HEAD`, `Branch` changes, or a referenced `PR #N` is closed/merged (`done`). Mark stale handoffs in state and pickup output instead of injecting outdated 5-field goals, and ensure `pre-compact` only preserves a non-stale handoff from the current session (replacing stale or foreign-session handoffs with a fresh current-HEAD auto-checkpoint).
- **#40 (`internal/state/state.go`, `internal/hook/hook.go`, `internal/mcpserver/server.go`):** Record the parent client PID (`os.Getppid()` when `> 1`) in `WriteWriterLease`, check OS process liveness (`IsProcessAlive`) so exited processes never trigger competing-writer warnings, and resolve `CurrentSessionID` in `ateam mcp` from env, `initialize` params, or a lease matching the MCP server's parent client PID.
- **#41 (`internal/state/state.go`, `internal/mcpserver/server.go`, `cmd/ateam/main.go`):** Drop cards not present in the remote forge snapshot during `MergeForgeCards`, record `board_synced_at` (`BoardSyncedAt`) on sync, print `synced=` in `BuildPickupSummary`, and perform a bounded best-effort remote forge sync on CLI/MCP `pickup` and `board` (falling back to cached state when offline).
- **#42 (`internal/state/state.go`, `internal/bootstrap/bootstrap.go`, `cmd/ateam/main.go`, `Taskfile.yml`):** Embed and read VCS build revision via `runtime/debug.ReadBuildInfo()` and `-ldflags`, include it in `ateam version` and `[ateam]` pickup headers, and add `ateam doctor` checks for binary-commit ancestry against `HEAD` and duplicate running `ateam serve` processes.

## Impact

- Scope: `cmd/ateam/main.go`, `internal/state/`, `internal/hook/`, `internal/mcpserver/`, `internal/bootstrap/`, `Taskfile.yml`, `docs/mcp.md`, and `openspec/`.
