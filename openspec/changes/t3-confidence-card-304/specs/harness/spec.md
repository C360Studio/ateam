## ADDED Requirements

### Requirement: T3 Code in-thread confidence card

The pickup and handoff skills SHALL, only when the T3 Code `html_render` tool is available and `T3CODE_THREAD_ID` is non-empty, render one card with `html_render` after their text report, built from `.agents/skills/handoff/t3-card.html`. The card SHALL show five rows (Proposal, Claim, Gates, Review, Merge), SHALL write `not checked` for any fact not read in that turn, and SHALL place agent-written handoff text only in a region labelled as not verified. The template SHALL contain no script and no remote URL and SHALL stay under 8 KB. On any other host the skills SHALL render nothing.

#### Scenario: Generated template stays static and small

- **WHEN** `ateam init` writes `.agents/skills/handoff/t3-card.html`
- **THEN** the file contains no `<script` and no `http`, is under 8,192 bytes, and contains the Proposal, Claim, Gates, Review and Merge rows and the not-verified region

#### Scenario: Paragraph only under the T3 condition

- **WHEN** `ateam init` writes the pickup and handoff skills
- **THEN** each contains exactly one card paragraph, which begins with the condition that `html_render` is available and `T3CODE_THREAD_ID` is non-empty and names `t3-card.html`
