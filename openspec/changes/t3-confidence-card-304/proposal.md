# Change Proposal: `t3-confidence-card-304`

Approved-by:

## Why

In T3 Code, someone following a thread on web or mobile sees only the agent's markdown and the PR's checks list (#302). Nothing in the thread itself shows how far to trust the work: whether the proposal is approved, who holds the lease, what the gates say, the review record, and the merge state. T3's `html_render` tool shows one self-contained HTML page inline above the agent's reply. Only the agent in that thread can call it, on its own turn. Pickup and handoff are the two turn boundaries ateam already owns, and each has just read the facts such a card needs (#304).

## What Changes

- The canonical pickup and handoff skills gain one short paragraph. In a T3 thread, after the text report, the agent renders a five-row card with `html_render`. The rows are Proposal, Claim, Gates, Review and Merge, filled from facts read this turn. A fact the agent did not check shows as `not checked`. Agent-written handoff text sits in a visibly labelled region.
- One shared card template, `.agents/skills/handoff/t3-card.html`, holds the layout: inline CSS, no script, no remote URL, under 8 KB, and legible at 360 px. A skill reads it only in a T3 thread, so other hosts pay no extra instruction text.
- The adopter templates in `internal/bootstrap/templates.go` carry the same paragraph and file, and `ateam doctor` parity covers the file.
- One spec Requirement covers the trigger, the five rows, the honesty rule, and the template limits.

## Track

`design`: it adds a spec Requirement and changes generated skill text that adopting repositories receive. See `design.md`.

## Non-goals

- No Go behaviour change: no `pickup --json` and no new command. If the card cannot be filled reliably from the printed report, `pickup --json` gets its own issue.
- No card on other hosts, no streaming, no T3 fork, nothing sent upstream to T3 (owner, 2026-11-06).
- It does not replace #302's commit statuses.

## Definition of Done

- The paragraph and template are in the canonical skills and the bootstrap templates, and the parity tests and `ateam doctor --check` are green.
- A bootstrap test checks that:
  - the paragraph appears only under the T3 condition;
  - the template has no `<script`, no `http`, and stays under 8 KB;
  - every row's placeholder is present.
- The owner renders the card once in a T3 thread on this repository, at pickup and at handoff, on web and mobile. The one-line result goes in `tasks.md`.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green. The change is archived before ready.
