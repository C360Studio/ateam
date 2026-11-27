# Proposal: decision-line-grammar-421
Approved-by: Coby Leuschke (2026-11-27) sha256:6b4d59375760

## Track

`design`: adds a harness Requirement for what counts as a recorded decision on an issue.

## Decisions you are making by approving

- An owner ruling written the way this repository writes them ("Owner decision" or "Owner ruling", with a date on the line) counts as the issue's recorded decision, so the queue check and the session-start summary see it.
- A line that starts the same way but carries no date, such as a question asking for an owner decision, does not count, so an issue still waiting on you is not reported as decided.
- A decision line written in bold counts the same as a plain one, for the new phrasing and the existing one.

## Why

ateam records an issue's decision only from a line starting `Decision:` or `Ruling:` (#416, design D4). Owner rulings here read `Owner decision 2026-11-26:` or `**Owner ruling (2026-10-30):**`, so the queue check's "decision recorded" finding and the session-start "Recent decisions" list miss every one of them (#421).

## What Changes

1. The decision-line match also accepts a line starting `Owner decision` or `Owner ruling` (so `Owner rulings` too) that contains a `YYYY-MM-DD` date.
2. Leading `*` and `_` are ignored before matching, for every prefix.
3. One ADDED harness Requirement states the grammar.

## Scope & Non-Goals

- **In-bounds:** `internal/state/forge.go` (`findDecisionLine`), its tests, `openspec/specs/harness/spec.md` (one ADDED Requirement).
- **Non-goals:**
  - a configurable prefix list;
  - telling who wrote a ruling (an agent records most of them on the owner's instruction);
  - the 120-byte truncation, the label fallback, and the newest-comment-first order, which stay as they are;
  - the board UI, which does not render the field.

## Definition of Done

- A table test with literal lines covers each accepted and rejected form, including the undated `Owner decision needed:` line, and `task mutate:check` detects removal of the date condition and of the emphasis strip.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before ready.
