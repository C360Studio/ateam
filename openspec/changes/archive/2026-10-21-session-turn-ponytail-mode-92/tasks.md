# Tasks: Session- and Turn-Scoped Ponytail Modes and Conditional AGENTS.md Ladder (`#92`)

- [x] 1. Add `SessionModes`, `SetSessionMode`, `EffectiveModeForSession`, and `EffectiveMode` to `state.State` and render `st.EffectiveMode()` in `BuildPickupSummary` (`internal/state/state.go`).
- [x] 2. Scope persistent `/mode` switches by `session_id` / `conversationId` when present and support ephemeral single-turn `[ponytail:<level>]` / `[mode:<level>]` overrides in `before-prompt` (`internal/hook/hook.go`).
- [x] 3. Support `--session <id>` in `ateam state mode` and session-scoped `ateam_state(action="mode")` (`cmd/ateam/main.go`, `internal/mcpserver/server.go`).
- [x] 4. Update `AGENTS.md`, `defaultAgentsMD`, `.agents/skills/ponytail/SKILL.md`, `docs/hook-flow.md`, and `openspec/specs/harness/spec.md` so `mode=off` skips the ladder and exit lines.
- [x] 5. Add unit and property/fuzz tests with `// spec:` attribution and verify with `task mutate:check`, `ateam doctor --check`, and `task verify`.
