# Tasks: Per-Agent, Per-Branch Handoff Slots (`#72`)

- [x] 1. Add `Handoffs map[string]*Handoff` (`<agent>@<branch>`) to `state.State` and alias `LastHandoff` to the current session's slot (`internal/state/state.go`).
- [x] 2. Apply `#39`/`#53` staleness rules per entry in `RefreshHandoffStaleness`, dropping stale entries from `Handoffs` on pickup (`internal/state/state.go`).
- [x] 3. Render current `<agent>@<branch>` handoff (`Last Handoff:\n...` or `Last Handoff: none for <agent>@<branch>`) followed by sorted `other handoffs:` lines in `BuildPickupSummary` (`internal/state/state.go`).
- [x] 4. Accept `--agent` / `agent` in `ateam state handoff`, `ateam state pickup`, and `ateam_state`, defaulting to hook `--agent`, `ATEAM_AGENT`, writer lease, or branch prefix (`cmd/ateam/main.go`, `internal/mcpserver/server.go`, `internal/hook/hook.go`, `internal/web/server.go`).
- [x] 5. Add unit and property/fuzz tests (`TestPerAgentBranchHandoffSlots`, `FuzzPerAgentHandoffSlotHistory`) and verify with `task mutate:check` and `task verify` (`internal/state/state_test.go`).
