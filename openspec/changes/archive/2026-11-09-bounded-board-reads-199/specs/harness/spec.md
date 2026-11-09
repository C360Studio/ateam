## ADDED Requirements

### Requirement: Bounded board reads

`ateam_state` `board` and `ateam state board` SHALL accept an optional `scope` (`open`, `done`, `all`), `limit` and `after` cursor, return cards ordered by number descending then ID, and report per-column `counts` over all cards, the matching `total`, `next_after` and `complete`, alongside the existing sync and fallback fields. Invalid arguments SHALL fail the call.

#### Scenario: Paging traverses each card once

- **WHEN** a client pages a mixed open/done board with any `scope` and `limit`, following `next_after` until `complete` is true, while card columns change between pages
- **THEN** the concatenated pages contain every card that matches `scope` throughout the traversal exactly once, in order, and no card twice

#### Scenario: Omitted history stays visible

- **WHEN** a response's `scope` excludes `done` cards
- **THEN** `counts.done` reports how many there are, and `scope: "done"` returns them

#### Scenario: Invalid arguments fail

- **WHEN** `scope` is not one of `open`, `done`, `all`, `limit` is negative, or `after` does not parse
- **THEN** the call fails and names the argument, returning no cards
