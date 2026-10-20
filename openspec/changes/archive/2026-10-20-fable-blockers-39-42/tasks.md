# Tasks: Remediate Fable dogfooding state, lease, board cache, and binary provenance blockers (#39-#42)

implemented-by: gemini

## 1. Handoff staleness (#39) and Writer lease session/PID liveness (#40)

- [x] 1.1 Mark handoffs stale on HEAD SHA drift, branch change, or closed/merged referenced PR in `internal/state/state.go`, omit stale 5-field goal blocks from active pickup, and prevent `pre-compact` in `internal/hook/hook.go` from re-saving stale or foreign-session handoffs.
- [x] 1.2 Record parent client PID in `WriteWriterLease`, check PID liveness (`IsProcessAlive`) before warning in `BuildPickupSummary`, and resolve session identity in `internal/mcpserver/server.go` so MCP `pickup` never warns on the current session.

## 2. Board sync & pruning (#41) and Build provenance & doctor checks (#42)

- [x] 2.1 Prune cards absent from the remote forge in `MergeForgeCards`, record and display `board_synced_at` (`synced=`), and perform bounded best-effort remote sync in CLI/MCP `pickup` and `board`.
- [x] 2.2 Report VCS build commit in `ateam version` and `[ateam]` pickup output, and check binary commit ancestry against `HEAD` plus duplicate `ateam serve` processes in `ateam doctor`.
- [x] 2.3 Add spec-attributed unit tests, property tests, and non-destructive `task mutate:check` detections across `internal/state`, `internal/hook`, `internal/mcpserver`, and `internal/bootstrap`, and pass `task verify` and `./bin/ateam doctor --check`.
