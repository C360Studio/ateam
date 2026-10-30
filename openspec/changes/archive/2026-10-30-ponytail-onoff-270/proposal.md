# Change Proposal: `ponytail-onoff-270`

Approved-by: Coby Leuschke (2026-10-30)

## Why

`LadderPrompt` (`internal/hook/hook.go:434-449`) emits the same ladder and guardrail lines for `lite`, `full`, and `ultra`; the three differ by one sentence. `AGENTS.md` carries the full ladder unconditionally, no doc or skill says when to pick `lite` over `ultra`, and the Mercury A/B (#58, PR #85) measured a token tax with no correctness gain. The owner ruled on 2026-10-30 (#270, closing epic #91 and #92-#94) that the OpenSpec `quick`/`design` track is the effort dial and Ponytail is on or off.

## What Changes

- `state.DisciplineMode` keeps two values, `on` and `off`; `ModeLite`, `ModeUltra`, and `ModeFull` go. `NormalizeMode` maps `on`, `""`, `full`, `lite`, and `ultra` to `on`, and `off`, `stop`, `normal` to `off`. The default stays on.
- `ateam hook` recognizes `/ponytail on|off` (and `/mode`), still accepting `full|lite|ultra`; `LadderPrompt` emits one level line under `PONYTAIL MODE [on]`.
- The CLI, MCP `mode` action, and `/api/mode` accept the same values through `NormalizeMode`; the MCP prompt text, CLI help, and bootstrap ponytail template name on/off.
- The board's discipline pill group becomes a two-state on/off control.
- The ponytail skill's levels table becomes on/off with the line "The OpenSpec quick/design track is the effort dial; Ponytail is on or off."; README and `docs/` follow.
- The harness spec's mode-switch scenario uses `on`/`off` and states the alias.

## Track

`quick`: no Requirement is added or removed, one capability's scenario changes, and no exported surface reachable from an adopting repository is added.

## Non-goals

- No per-session or per-turn scoping (#92), subagent propagation (#93), or classifier (#94).
- The hook reminder itself stays; whether `on` earns its ~110 tokens is #95's question.
- No change to the `State.Mode` JSON key or the `/api/mode` and MCP `mode` shapes; only accepted values change.
- Removing the `lite`/`ultra`/`full` aliases is a follow-up issue for the release after this one.

## Definition of Done

- `ateam state mode lite` and `/ponytail ultra` resolve to `on` and say so; `ateam state mode banana` is still rejected.
- A table test on `NormalizeMode` covers every alias and an unknown value, and `task mutate:check` reports `detection` with the `lite`/`ultra` alias arms removed.
- `task verify`, `ateam doctor --check` (MCP schema under 300 tokens), and `ateam bench --selftest` green; the change archived before ready.
