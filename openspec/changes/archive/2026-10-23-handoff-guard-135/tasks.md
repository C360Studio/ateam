## 1. Implementation

- [x] 1.1 Share one held-slot predicate (`Handoff.HoldsSlotAgainst`, `state.AutoSaveGoal`) between pre-compact and `State.SetHandoffGuarded` in `internal/state` (#135).
- [x] 1.2 Refuse on the CLI (exit 2, `--force`), MCP (tool error, `force`), and HTTP (409, `?force=1`) paths; the refusal returns before any PR comment or writer-lease clear (#135).
- [x] 1.3 Tests: `TestSetHandoffGuardedRefusesAnotherSessionsFreshHandoff` (different, unknown writer, unknown holder, forced, same session, empty, stale, auto-save), `TestStateHandoffRefusesAnotherSessionsFreshHandoff`, `TestMCPHandoffRefusesAnotherSessionsFreshHandoff`, `TestServerEndpoints` 4b. `task mutate:check` mutants, all `detection`: guard off, unknown session allowed, auto-save treated as held (`state.go` sha256 `975f1460`); CLI exit 2 removed (`main.go` `ee7c1f74`); HTTP 409 removed (`web/server.go` `31e04065`); MCP always forced (`mcpserver/server.go` `8fff098e`) (#135).
