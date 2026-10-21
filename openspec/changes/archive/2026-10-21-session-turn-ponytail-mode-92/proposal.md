# Change Proposal: Session- and Turn-Scoped Ponytail Modes and Conditional AGENTS.md Ladder (`#92`)

## Why
A single checkout-wide `State.Mode` in `.ateam/state.json` causes mode switches in one session or subagent to clobber concurrent agents sharing the checkout, and unconditional ladder rules in `AGENTS.md` keep the full 6-step ladder and 2-line exit summary active even when `mode = off`.

## What Changes
- Add `SessionModes map[string]DisciplineMode` (`session_modes,omitempty`) to `state.State` with deterministic precedence: ephemeral turn tag (`[ponytail:<level>]` / `[mode:<level>]`) > `session_modes[session_id]` > `ATEAM_MODE` > checkout `mode` (`full` default).
- Scope `/ponytail <level>`, `/mode <level>`, `ateam_state(action="mode")`, and `ateam state mode --session <id>` to the active `session_id` / `conversationId` when present while keeping checkout-wide fallback when no session ID is supplied.
- Support ephemeral single-turn `[ponytail:lite|full|ultra|off]` / `[mode:...]` tags in `before-prompt` without mutating `.ateam/state.json`.
- Update `AGENTS.md`, `defaultAgentsMD`, `.agents/skills/ponytail/SKILL.md`, and `docs/hook-flow.md` so `mode = off` skips the 6-step decision ladder and 1–2 line exit summary while preserving repository verification and trust-boundary guardrails.

## Impact
- `internal/state/state.go`, `internal/state/state_test.go`
- `internal/hook/hook.go`, `internal/hook/hook_test.go`
- `cmd/ateam/main.go`, `internal/mcpserver/server.go`
- `AGENTS.md`, `internal/bootstrap/templates.go`, `.agents/skills/ponytail/SKILL.md`, `docs/hook-flow.md`, `openspec/specs/harness/spec.md`
