# Change Proposal: `bounded-board-reads-199`

Approved-by: Coby Leuschke (2026-11-09)

## Why

The MCP `board` action returns every cached card with no filter or limit (#199). At `11e8871` on this repository, `ateam state board` printed 54,944 bytes for 122 cards. 100 of those cards are `done` and account for 43,220 bytes, while the 22 actionable cards take 8,282. An agent asking for its next work pays for the history every time. `ateam doctor`'s 230-token schema figure covers discovery only, never this response.

## What Changes

- `ateam_state` `board` takes three optional arguments. `scope` is `open`, `done` or `all`, `limit` is a card count, and `after` is a continuation cursor. `ateam state board` gains the matching `--scope`, `--limit` and `--after` flags.
- Cards come back in one documented, deterministic order, and pages are keyed on immutable card fields. A full traversal therefore never loses or repeats a card that exists throughout it.
- The response gains additive fields: per-column `counts` over all cards (so a filtered-out `done` total is always visible), `total` for the matching set, and `next_after` plus `complete` for continuation. The existing `forge`, `synced_at`, `cached_fallback`, `sync_error`, `notice` and `cards` fields keep their meaning.
- A new spec Requirement, plus `docs/mcp.md` rows for the arguments and fields.

## Track

`design`: this adds a spec Requirement and changes the MCP tool schema and response, which are exported surface reachable from adopting repositories. See `design.md`.

## Non-goals

- Board UI or `/api/cards` changes.
- Changing the forge sync's closed-item cap (`internal/state/forge.go:445`).
- A new merge gate, or any context-token guarantee.
- Instruction-reading cost (#200) and mode resolution (#92).

## Definition of Done

- For a mixed open/done fixture and every `limit`, following `next_after` from the first page yields exactly the reference model's filtered and sorted cards: no loss, no duplicates, in order. A generated check (`testing.F`) covers the interacting `scope` × `limit` × `after` inputs.
- `scope: "done"` returns completed history, and every response reports `counts` including the `done` total.
- Calls that set none of the three arguments get the default the owner rules on (design OQ1). That default is documented in `docs/mcp.md`.
- Before/after response bytes on this repository are recorded in `tasks.md`, separate from the schema token figure. The schema stays under 300 tokens.
- `task mutate:check` reports `detection` for the scope filter, the cursor comparison, and the `complete` flag.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green. The change is archived before the PR is marked ready.
