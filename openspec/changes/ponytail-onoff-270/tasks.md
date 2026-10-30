## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` (#270). Evidence: the owner approved in Enjoy on 2026-10-30 (ruling recorded on #91–#94 and #270); Claude wrote the header at their request.

## 2. State and hook

- [ ] 2.1 `DisciplineMode` keeps `ModeOn` and `ModeOff`, and `NormalizeMode` maps `on`, `""`, `full`, `lite`, `ultra` to on and `off`, `stop`, `normal` to off, unknown values to `(ModeOn, false)` (`internal/state/state.go:442-465`); the load fallback (`state.go:1528`), `defaultState`, and `internal/bench/bench.go:664` use `ModeOn`. Acceptance: a table test on `NormalizeMode` covers every alias, case and whitespace, and an unknown value; `task mutate:check` reports `detection` with the `lite`/`ultra` arms removed (#270).
- [ ] 2.2 `modeCommandRe` accepts `on|full|lite|ultra|off` (`internal/hook/hook.go:61`), `LadderPrompt` loses its `switch` and emits one level line under `PONYTAIL MODE [on]` (`hook.go:434-449`), and `detectModeSwitch` falls back to `ModeOn` (`hook.go:430`). Acceptance: a test on `detectModeSwitch` and `LadderPrompt`; `TestBeforePromptInjectsLadderOnlyOnModeSwitch` sends `/ponytail ultra` and reads `[on]` (#270).

## 3. Entry points, skill, and UI

- [ ] 3.1 The MCP prompt text (`internal/mcpserver/server.go:416`), CLI help (`cmd/ateam/main.go:184`), and bootstrap ponytail template (`internal/bootstrap/templates.go:436-437`) name on/off; the canonical skill `.agents/skills/ponytail/SKILL.md` and its Claude adapter follow and gain the effort-dial line. Acceptance: `ateam doctor --check` green with the MCP schema under 300 tokens (#270).
- [ ] 3.2 The board's discipline pill group becomes an on/off control with the same `/api/mode` body (`ui/src/routes/+page.svelte:1121-1134`); `ui/src/lib/board.test.ts` fixtures use `on`, and `internal/web/server_test.go` posts `ultra` and reads `on`. Acceptance: `task ui:check` and `task ui:test` pass (#270).

## 4. Docs and spec

- [ ] 4.1 `README.md` (4 mentions), `docs/hook-flow.md:59-63`, `docs/adoption.md:145`, `docs/mcp.md:100`, and `docs/daily-use.md:9,19,44` name on/off; `docs/hook-flow.md` says `lite`, `ultra`, and `full` are aliases of `on` for this release only. The spec delta rewrites the mode-switch scenario of the harness hook requirement (`openspec/specs/harness/spec.md:18-21`). Acceptance: `task spec:check` passes (#270).

## 5. Verify

- [ ] 5.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0; `ateam state mode lite` prints `on` and `ateam state mode banana` exits 1 (#270).
