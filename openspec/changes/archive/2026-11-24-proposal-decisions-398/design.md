# Design: proposal-decisions-398

## Context

- `internal/state/openspec.go:57,60` — `maxProposalLines` (60) and `maxDesignLines` (120) are spec-check's only reviewability limits; `:601-606` prints them as warnings, exit 0.
- `internal/state/openspec.go:317` — `ScanOpenSpecQueue` skips `archive`, so every check here sees open changes only; `:344,353` read `proposal.md` and `design.md` through `ParseProposal` (`:64`), which returns the `Approved-by:` value and a line count.
- `internal/state/openspec.go:585-599` — the stale-approval branch is the existing pattern for a spec-check *failure* (exit 1) after the warnings are collected.
- `internal/state/openspec.go:619-700` — `ParseSpecMarkdown` parses canonical specs only: a section counts as requirements when its heading starts `requirements` (`:655`), so a delta's `## ADDED Requirements` is ignored; scenario steps are the `- ` lines (`:691-695`).
- `node_modules/@fission-ai/openspec/dist/core/validation/validator.js:266-276` — OpenSpec 1.14.3 warns on an ADDED Requirement over 500 characters (strict mode fails on it) and leaves MODIFIED alone; `:733` applies the same limit to canonical specs, so a long MODIFIED Requirement first fails after archive (PR #362, #366). The global OpenSpec 1.7.0 that `spec:check` falls back to checks neither.
- No check counts scenario words; no check reads proposal sections. `git log -S maxProposalLines` → `2d7eeef` (#142), `ceaae4a` (#178).
- Generated text: `internal/bootstrap/templates.go:294` (protocol § 2 step 1), `:336-348` (`openspecConfig`, only `rules: design:`), `:349` (`architectContract`), `:361-385` (design skill). Repository copies: `.agents/protocol.md:30`, `.agents/contracts/architect.md`, `.agents/skills/design/SKILL.md`, `openspec/config.yaml`.
- Open change in flight elsewhere: `openspec/changes/gemini-neon-hydrogen-405/` (draft PR #406) has a `design.md` and no decisions section.

## Goals / Non-Goals

**Goals:** a short plain-language place in every proposal where the owner sees what they are deciding; early notice of spec text too long to review.
**Non-Goals:** judging honesty or completeness of the list; checking archived changes; a parser for the `## Track` section.

## Decisions

### D1. How spec-check finds the section

- **A (recommended): one fixed heading**, `## Decisions you are making by approving`, matched case-insensitively, bullets until the next `## `. Cost: a few lines beside `ParseProposal`. Forecloses: authors' own heading wording.
- **B: any heading starting `## Decisions`.** Looser; also matches a design-style "Decisions" section pasted into the proposal, which is the detail the owner should not have to read.
- *Adopter seam:* the heading text is the contract; the protocol, design skill and `rules: proposal:` all quote it. Doing nothing: an adopter's existing proposals warn until they add it.

### D2. Warn or fail, and how the track is known

- **A (recommended): fail when the change has a `design.md`, warn otherwise.** `design.md` is the only track signal spec-check can read without a new parser; a quick change never has one (protocol § 2 step 1). Blast radius: any open design-track change fails `task verify` until it adds the section — today #406.
- **B: warn on both tracks.** No migration cost, but the owner of a design change can still approve with nothing listed, which is the #398 failure.
- **C: fail on both tracks.** Strongest; breaks every quick change in adopter repos at upgrade.
- *Adopter seam:* upgrading ateam makes an open design change red until its proposal gains the section; the failure message names the heading to add.

### D3. What a bullet must contain

- **A (recommended): at least one bullet; warn on a bullet with fewer than 5 words left after removing code spans and numbers.** Catches "- `MaxQueue` = 64" without judging prose. Warning only, since word counts misfire.
- **B: presence only.** Simplest; lets an identifier list pass as decisions, which #398 rules out.
- *Adopter seam:* "None: this change alters no behaviour a user sees." is a valid bullet.

### D4. Long Requirement and scenario text

- **A (recommended): spec-check warns on a delta Requirement body over 500 characters (ADDED and MODIFIED) and a scenario step over 60 words.** 500 matches OpenSpec's own unit, so the same text passes or warns in both tools, and MODIFIED is caught before archive. Reuses `ParseSpecMarkdown` by also treating `ADDED`/`MODIFIED Requirements` headings as requirement sections. Cost: one heading match plus a walk over `openspec/changes/*/specs/**/spec.md`.
- **B: a word limit per sentence, as #398 sketches.** Needs sentence splitting that backticks, `e.g.` and paths defeat; a second unit beside OpenSpec's characters.
- **C: rely on OpenSpec.** Misses MODIFIED deltas, scenarios, and the 1.7.0 fallback.
- *Adopter seam:* warnings only; nothing fails for length.

### D5. Where the guidance lives

- **A (recommended):** protocol § 2 step 1 lists the section among the proposal's contents; the design skill and architect contract say the proposal states the decisions the design makes; `openspec/config.yaml` gains `rules: proposal:` so `openspec instructions proposal` shows it. Each in repository copy and template; an existing adopter `config.yaml` stays byte-for-byte unchanged (current init behaviour).
- **B: protocol only.** Agents drafting with `openspec instructions` never see it.

## Risks / Trade-offs

- A section can be present and still miss the decision that matters (#398 *Limits*); this is a place to notice absence, not a proof.
- D2 A turns #406 red once both are on `main`; whichever lands second adds the section.

## Migration Plan

Archived changes are never scanned. Open changes: this one dogfoods the section; #406 adds it if it rebases after this lands.

## Open Questions

1. **Owner ruling:** D2 — fail on design track (A) or warn everywhere (B)?
2. **Owner ruling:** thresholds — 5 plain words per bullet, 500 characters per Requirement, 60 words per scenario step.
3. **Owner ruling:** the exact heading text in D1.
