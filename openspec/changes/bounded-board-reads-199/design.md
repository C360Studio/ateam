# Design: `bounded-board-reads-199`

## Context

Inventory taken at `11e8871`. Every line below was opened in this session.

- `internal/state/state.go:2266-2274` declares `BoardSnapshot` (`forge`, `synced_at`, `board_synced_at`, `cached_fallback`, `sync_error`, `notice`, `cards`). `BoardView` (`:2278-2303`) copies `st.Cards` unfiltered and unsorted, then adds the stale-cache notice.
- There are three callers of `BoardView`, all of which `json`-encode the full snapshot: the MCP tool (`internal/mcpserver/server.go:400-402`), the MCP resource `ateam://state/board` (`:501-503`), and `ateam state board` (`cmd/ateam/main.go:487-498`). `forge_test.go:272` reads only its sync metadata. The UI reads `/api/cards` (`internal/web/server.go:471`) and never sees `BoardView`.
- Card order today is merge order (`MergeForgeCards`, `state.go:1940-1977`): existing cards keep their slots and newly synced cards are appended. Nothing documents it, and it shifts across syncs.
- Card IDs are `<prefix>-<number>` (`forge.go:796`, e.g. `GH-354` and `GH-PR-247`) and never change for an item. `Column` does change: an issue moves to `done` when it closes. Local cards use `ATEAM-` IDs and may have `Number` 0 (`state.go:1965`).
- The forge sync keeps at most 50 closed issues and 50 closed PRs (`forge.go:444-445`), so `done` history is bounded at about 100 cards per forge while open items are never capped.
- The `ateam_state` arguments are decoded at `server.go:255-274`. A wrong-typed field already fails the call (#266). Schema: `server.go:51-76`. `ateam doctor` reports 919 bytes / 230 tokens against the `<300` budget, which leaves about 280 bytes.
- Spec: `openspec/specs/harness/spec.md:128` requires board responses to surface the sync timestamp and fallback status. Docs: `docs/mcp.md:103` and `:137`.
- Measured with `ateam state board` here: 54,944 bytes and 122 cards in total; `done` is 100 cards and 43,220 bytes; open is 22 cards and 8,282 bytes (6,823 compact).

## Goals / Non-Goals

Goal: an agent can ask for actionable work at a bounded cost, explicitly page through history, and always see what was left out and how fresh the cache is. Non-goals: see `proposal.md`.

## Decisions

### D1. Filter, paginate, or both

- **A (recommended): `scope` plus keyset `limit`/`after`.** `scope` filters by column (`open` means every column except `done`). `limit` caps the page size and `after` resumes from a page. The filter alone fixes today's case, but open items have no cap (`forge.go:444`), so only a limit bounds the response. This costs about 60 lines in `state` plus the argument plumbing. It forecloses nothing.
- **B: `scope` only.** About 20 lines, but `scope: "done"` still returns about 43 KB and open work is still unbounded.
- **C: `limit`/`after` only.** This is bounded, but an agent pages through history to find open work, which defeats the point.

### D2. Order and cursor

- **A (recommended): order by `Number` descending, then `ID` ascending.** Both are immutable per card. The cursor is an opaque `<number>:<id>` taken from the last returned card, and the next page starts strictly after that key. A card that exists for the whole traversal is returned exactly once, even if it changes column or the cursor card disappears between pages. A card created mid-traversal with a number below the cursor is missed until the next read, which matches what offset cursors already accept.
- **B: an offset cursor.** This is simpler, but any sync between pages shifts positions and loses or repeats cards. That is exactly what #199 asks us to prevent.
- **C: column first, then time.** It reads nicely, but `Column` and `UpdatedAt` both change, so the keyset breaks.
- Sorting applies to every `BoardView` response, including unfiltered ones. **Adopter seam:** a consumer that relied on merge order sees a different order but the same set and fields. Nothing in this repository relies on it.

### D3. Response shape

Fields are additive on `BoardSnapshot`. `scope` is the effective scope; `counts` is cards per column over all cards, before filtering; `total` is the number of cards matching `scope`; `next_after` is empty on the last page; `complete` is `true` when `cards` holds every match. Existing fields and the stale-cache `notice` are unchanged. **Adopter seam:** a decoder that ignores unknown fields keeps working. A strict decoder breaks, and none exists here.

### D4. Validation

An unknown `scope`, a negative `limit`, or an `after` that does not parse fails the call (`isError`) and names the argument. It is never silently treated as "all". `limit` 0 or omitted means no limit.

### D5. Surfaces

The MCP tool and `ateam state board --scope/--limit/--after` share one `state.BoardQuery`. The MCP resource takes no arguments and keeps the unfiltered view, now with `counts`. The schema adds about 95 bytes (`scope` as a 3-value enum, `limit` as an integer, `after` as a string). Task 2.3 measures the result; the estimate is about 255 tokens.

## Risks / Trade-offs

- **Default (OQ1).** Keeping `all` as the omitted-argument default is safe for every existing consumer, but saves nothing until callers pass `scope`. Defaulting the MCP tool to `open` saves about 85% by default. In return it changes an existing caller's result, though `counts.done` and a `notice` make the omission explicit, never silent.
- **Sort cost.** Sorting is O(n log n) over about 220 cards at most, which is negligible next to the forge sync.
- **Missed new low-numbered cards** (D2 A) are accepted. Forge numbers increase over time, so new items sort first, and a re-read from the top sees them.

## Migration Plan

1. Ship the arguments, flags, fields and order in v0.4.0, under whichever default OQ1 picks.
2. If OQ1 picks `open` for the MCP tool, `docs/mcp.md` and the release notes call it out. A caller that wants the old result passes `scope: "all"`.

## Open Questions

1. **OQ1 (owner ruling):** what should an MCP `board` call with no `scope` return? (a) `all`, unchanged for existing callers; (b) `open` on the MCP tool only, with `counts` and a `notice` naming the omitted `done` total, while the CLI and resource keep `all` (recommended, because agents are the measured cost and humans and scripts keep the full view); (c) `open` everywhere.
2. **OQ2 (owner ruling):** should the MCP tool also apply a default `limit` when none is given? Recommendation: no. Open work here is 22 cards and filtering already cuts about 85%. Revisit it with a `// shortcut:` note if open boards grow past about 100 cards.
