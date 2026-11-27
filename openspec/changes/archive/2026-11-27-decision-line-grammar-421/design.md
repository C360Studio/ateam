# Design: decision-line-grammar-421

## Context

- `internal/state/forge.go:884-896`: `findDecisionLine` trims each line and matches `decision:` or `ruling:` on the lowercased start, then truncates to 120 bytes. No other prefix and no markdown emphasis is accepted.
- `forge.go:865-882`: `extractDecisionSummary` scans comments newest first, then the body, then labels starting `decision:`, and fills `BoardCard.Decision`.
- Readers of `BoardCard.Decision`: `internal/state/queuecheck.go:237` (Q3, "needs-decision, but a decision is recorded"); `internal/state/state.go:3168-3181` (the session-start "Recent decisions" list, first five cards); `state.go:1972-1977` (merge keeps an earlier decision). The board UI declares `decision` (`ui/src/lib/board.ts:15`) but no component renders it.
- No spec Requirement states the grammar today (`openspec/specs/harness/spec.md` has none; the queue-check Requirement at `:1377` says only "a recorded decision"). No doc names it.
- Observed 2026-11-27 on this repository's issue comments, issue bodies and PR bodies (leading `*`, `_`, `>` stripped): 86 lines start `Owner decision` or `Owner ruling(s)`, and 81 carry a `YYYY-MM-DD` date (`Owner decision 2026-11-26:`, `**Owner ruling (2026-10-30):**`, `Owner rulings on #415's open questions (2026-11-27, …`, `Owner decision, 2026-10-22:`). Of the five undated lines, `Owner decision requested: accept the ADR …` is a question, `Owner rulings are on #143.` and `Owner decisions in the text:` are pointers, and two are rulings (`Owner decision on #240: …`, `Owner rulings** (#130 comment): …`). Seven sampled comment rulings are bold.
- #416 design D4 ruled option A (keep the grammar) there and filed the widening as this issue.

## Goals / Non-Goals

**Goals:** this repository's owner rulings count as recorded decisions for Q3 and "Recent decisions", without making an owner *question* count as one.

**Non-Goals:** configurable prefixes, authorship checks, changing truncation, order or the label fallback, rendering the field in the UI.

## Decisions

### D1. How an owner ruling is recognised

- **A: prefix only** (`Owner decision`, `Owner ruling`, any continuation). Smallest change. But a needs-decision issue whose body says `Owner decision needed: pick A or B` would then read as decided, and Q3 reports exactly the issues still waiting on the owner: the false positive lands where the check matters.
- **B (recommended): prefix plus a `YYYY-MM-DD` date anywhere on the line.** One extra regexp. 81 of 86 observed lines carry a date; of the five that do not, three are a question or pointers and two are rulings this option misses. Forecloses undated rulings in that phrasing; they still match by starting `Decision:` or `Ruling:`.
- **C: a prefix list in `.agents/ateam.json`.** Lets an adopter add its own phrasing, but adds a setting, its validation and docs for one caller, with no adopter asking.
- *Adopter seam:* an adopter whose rulings use neither phrasing sees no change. One that writes `Owner decision 2026-…` lines now sees them as recorded decisions in Q3 and session start. Doing nothing is safe: every effect is advice.

### D2. Markdown emphasis

- **A (recommended): strip leading `*` and `_` before matching, for every prefix**, so `**Owner ruling (2026-10-30):**` and `**Decision:** X` both match. The stored line keeps its emphasis as written.
- **B: plain lines only.** Misses the bold rulings sampled above.
- Blockquotes (`>`) stay unmatched under both: a quoted ruling is a citation of an earlier one, and the original line already matches where it was written.
- *Adopter seam:* an adopter's bold `**Decision:**` lines start to count. That widens the existing grammar too, which is why it is a decision and not a detail.

## Risks / Trade-offs

- Milestone-placement notes (`Owner decision 2026-11-26 …: placed in v0.4.7`) now count. On a needs-decision issue, Q3 would report it as decided when the label was about another question. Advice only; the finding text quotes the line, so a reader sees which decision it was.
- "Recent decisions" lists the first five cards with a decision, so placement notes can crowd it. Unchanged ordering; a follow-up can rank by kind if it proves noisy.
- A date in an unrelated `Owner decision …` sentence (`Owner decision deadline 2026-12-01?`) still matches. No such line in the sample; accepted.

## Migration Plan

None: the field is recomputed on the next `ateam state sync`.

## Open Questions

- D1 and D2 need owner rulings (approving the proposal accepts the recommended options; name another option to change the spec delta and tasks before implementation).
