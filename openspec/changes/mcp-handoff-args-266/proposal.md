# Change Proposal: `mcp-handoff-args-266`

Approved-by: Coby Leuschke (2026-10-30)

## Why

Through MCP, `ateam_state(action="handoff")` saves a handoff with a missing, blank, or numeric `goal`. It replaces a useful checkpoint in the same session's slot with empty fields, and returns `isError: false` (#266). `internal/mcpserver/server.go:273` discards the `json.Unmarshal` error, so `"goal": 123` decodes to `""`. The handoff branch (`:305`) never checks the goal, which the CLI already requires (`cmd/ateam/main.go:319`). The schema requires only `action`, so nothing tells a client the goal is needed. `execTaskTool` (`:395`) discards its decode error the same way.

## What Changes

1. **A bad decode fails the call.** `execStateTool` and `execTaskTool` return `invalid arguments: <decode error>` with `isError: true` before anything else runs, including `read_handoff`.
2. **A handoff needs a nonblank goal.** Before the refresh, the cache write, the lease release, or the forge post, a handoff whose `goal` is missing or only whitespace returns `handoff: goal is required` with `isError: true`, matching the CLI's wording. The other actions are unchanged.
3. **The schema says so.** `goal` gains `"description": "handoff: required"`. That's about 8 tokens, on top of today's 222 (budget < 300). `required` stays `["action"]` so the other actions don't need a goal.
4. Spec: a scenario added to the harness Requirement "Two-tool MCP context budget".

## Track

`quick`. One package (`internal/mcpserver`) plus one schema description. It adds a scenario but no Requirement, adds no exported surface, and touches no boot path, durability behavior, or gate file.

## Non-goals

- Requiring other handoff fields, changing the `post` default, or adding a confirmation step.
- Foreign lease release (#265) and other-session slot protection (#135).
- Per-action JSON Schema (`oneOf`). That would blow the token budget.

## Definition of Done

- Before each invalid request, an MCP test seeds a useful same-session checkpoint and a writer lease, with a forge stub that counts post calls. The invalid requests are a missing, empty, whitespace-only, or numeric `goal`, and a numeric `post`. Each returns `isError: true`, leaves the cache and `writer.json` byte-identical, and makes zero post calls, including when `post` is omitted.
- A valid goal still replaces the same-session checkpoint, with the optional fields behaving as they do today.
- A wrong-type `target` passed to `ateam_task` returns `isError: true` and runs nothing.
- `task mutate:check` reports `detection` with the goal check removed, and again with the decode check removed. Examples are enough: each check takes one input and has no history.
- `TestMCPTokenBudget` stays under 300 tokens. `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
