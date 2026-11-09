## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1 (omitted-argument default) and OQ2 (default limit) (#199).
  - Evidence: the owner approved in Enjoy on 2026-11-09. OQ1 is (b): with no `scope`, the MCP tool returns `open`, with `counts` and a `notice` naming the omitted `done` total, while the CLI and the resource keep `all`. OQ2 is no default limit (revisit past about 100 open cards). Claude wrote the header at their request.

## 2. Bounded board read

- [x] 2.1 Add `state.BoardQuery{Scope, Limit, After}` and have `BoardView` take it (D1 A, D2 A, D3, D4).
  - Premise: `BoardSnapshot` is at `internal/state/state.go:2266-2274`, `BoardView` at `:2278-2303`, and the card ID format is set at `forge.go:796`.
  - Acceptance: tests carry `// spec: openspec/specs/harness/spec.md#Requirement: Bounded board reads`. A `testing.F` fuzz target builds mixed open/done card sets and checks every `scope` × `limit` combination against a reference model (filter, then sort by `Number` descending and `ID` ascending): concatenating pages via `next_after` gives the model's list exactly once, in order. Card columns are mutated between pages to prove keyset stability. Table cases reject an unknown scope, a negative limit and a malformed cursor. `counts` always includes `done`.
  - `task mutate:check` reports `detection` for dropping the scope filter, `>` → `>=` in the cursor comparison, and forcing `complete` to true.
  - Evidence: `FuzzBoardViewPaging` compares every page exactly against a reference model, with columns flipped between pages. The model is the cards matching scope now, strictly after the last returned card in the fixed order, capped at `limit`. It also checks `total`, `counts.done` and `complete` per page. The first version checked order, uniqueness and eventual inclusion only. Codex's review of `2148312` showed that a continuation-only scope leak survived it; the oracle, a matching seed and a table fixture (open #3, done #2, open #1) were added in response. `FuzzBoardViewPaging` and `TestBoardViewQuery` pass with `-race`, and a 20 s fuzz run found nothing. The scope-leak mutant (`q.After != "" ||`) now gives `detection` under both tests. `task mutate:check` verdicts are `detection` under both tests for dropping the `done` scope case, making the cursor comparison inclusive (`!k.before(after)`), and forcing `complete` true. `BoardView` now returns `(BoardSnapshot, error)`. Its four call sites are updated, including `forge_test.go:272`.

- [x] 2.2 Plumb the query through the MCP tool (`internal/mcpserver/server.go:255-274`, `:400-402`) and `ateam state board` (`cmd/ateam/main.go:487-498`) with the OQ1 default. The resource (`:501-503`) stays unfiltered (D5).
  - Acceptance: `server_test.go:153-154` style cases cover `scope: "done"`, a bad scope returning `isError`, and the omitted-argument default. A `main_test.go` case covers the CLI flags.
  - Evidence: `TestMCPBoardScope` covers the default (`open`, plus a notice that also keeps the stale-cache warning), `scope: "done"`, a bad scope (`isError`, no cards) and the unfiltered resource. Removing the MCP `open` default gives `detection`. `TestCLISubcommands` covers `--scope done --limit 1` (exit 0) and `--scope closed` (exit 1).

- [x] 2.3 Add `scope`, `limit` and `after` to the schema (`server.go:51-76`).
  - Acceptance: `ateam doctor` reports under 300 schema tokens. Record the before (230) and after figures here.
  - Evidence: `ateam doctor` went from 919 bytes / 230 tokens before to 1,027 bytes / 257 tokens after.

## 3. Docs and evidence

- [x] 3.1 Update `docs/mcp.md:103` and `:137` with the arguments, the fields, the order and the default.
  - Evidence: the `board` row and the resource row now cover the arguments, the fields, the order and the two defaults.

- [x] 3.2 Record `ateam state board` response bytes on this repository before (54,944 total, 8,282 open, 43,220 done) and after for the default, `scope: "open"` and `scope: "done"`, kept separate from the schema figure.
  - Evidence: both binaries were measured against the same 123-card board on 2026-11-09 (100 `done`, 22 backlog, 1 in progress). Before, the `main` binary's MCP `board` returned 55,405 bytes. After: the MCP default (`open`) returns 9,618 bytes, an 83% cut. `ateam state board --scope all` returns 55,536, `--scope open` 9,536 and `--scope done` 46,262. The schema figure is in 2.3, kept separate. The proposal's 54,944 was the 122-card board at `11e8871`.

## 4. Close

- [ ] 4.1 `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green on a clean tree, and a not-weaker review record is posted at the final head.
- [x] 4.2 Archive this change before marking the PR ready.
  - Evidence: archived before review, so the new Requirement resolves for `task test:spec`.
