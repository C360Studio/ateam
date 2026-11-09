## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1–OQ3 (#304). Evidence: the owner answered in Enjoy on 2026-11-09 with OQ1 one shared `t3-card.html` (D1 A), OQ2 tool and env var (D2 A), and OQ3 the owner renders it on web and mobile. Claude wrote the header at their request.

## 2. Card template

- [x] 2.1 Add `.agents/skills/handoff/t3-card.html` and its mirror in `defaultTemplateFiles()` (`internal/bootstrap/templates.go:597`) (D1 A, D3).
  - Premise: the layout follows T3's `html_render` limits (`tools.ts:78-104` @ `4f0b74e`).
  - Acceptance: a bootstrap test (`// spec: …#Requirement: T3 Code in-thread confidence card`) asserts that the generated file:
    - has no `<script` and no `http`;
    - is under 8 KB;
    - contains the five row placeholders and the "Agent-written (not verified)" region.
  - `task mutate:check` reports `detection` when the size bound is removed and when the `<script` check is removed.

  - Evidence: `t3-card.html` is 2.6 KB. Measured in headless Chromium with sample values: 551 px tall at 360 px wide, 378 px at 728, so the height is 560. `TestT3CardTemplate` checks the generated card and that the repository copy equals `t3CardHTML`; `TestT3CardViolations` checks the checker itself. Both pass with `-race`. `task mutate:check` reports `detection` for a `<script>` added to `t3CardHTML`. The size-bound mutant is in the test-only checker, which `ateam mutate` refuses (`-file must be a non-test .go source file`), so it ran as `go test -overlay` and `too_big` failed (`t3card_test.go:69`). Deviation: `ateam doctor` does not check the file, so the approved design and proposal were corrected.
## 3. Skill paragraphs

- [x] 3.1 Add the D2 A trigger paragraph to the canonical pickup § 3 (`.agents/skills/pickup/SKILL.md:29-37`), to handoff § 2, and to their adopter mirrors (`templates.go:399-448`).
  - Acceptance: a bootstrap test asserts that each generated skill contains the paragraph exactly once, that it starts with the T3 condition, and that it names `t3-card.html`. The existing parity tests pass unchanged for adapters.
  - `task mutate:check` reports `detection` when the paragraph is dropped from one template.

  - Evidence: `t3CardParagraph(kind, link)` and `t3CardCondition` are in `templates.go`. Both the generated skills and the repository's canonical skills carry the paragraph exactly once, with `html_render` named twice. `task mutate:check` reports `detection` for dropping the pickup paragraph (`card paragraph appears 0 times`). `ateam doctor --check` exits 0.
## 4. Docs

- [x] 4.1 Add one sentence to the `docs/adoption.md` T3 section (`:205-220`) naming the card and its trigger.
  - Acceptance: a reviewer reads it.

  - Evidence: one sentence after the `runOnWorktreeCreate` line in `docs/adoption.md` § T3 Code hosts. Reviewer read pending.
## 5. Hand verification (OQ3)

- [ ] 5.1 In a T3 Code thread on this repository, run `/pickup` and `/handoff` once each, then view the card on web and on the mobile app. Record a one-line result per surface here.

## 6. Close

- [ ] 6.1 `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green on a clean tree. A not-weaker review record is posted at the final head.
- [ ] 6.2 Archive this change before marking the PR ready.
