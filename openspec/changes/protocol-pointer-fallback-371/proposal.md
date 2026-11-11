# Change Proposal: `protocol-pointer-fallback-371`

Approved-by:

## Why

Since #200 (v0.4.0), the role contracts and the pickup, handoff and preflight skills tell the agent to read only numbered protocol sections, such as "read § 1 and § 2 step 5 (Close)". A brownfield adopter keeps its own `.agents/protocol.md`, which has no `## 1.` to `## 4.` headings, while `ateam init` copies the contracts byte for byte. The pointers then lead nowhere, and an agent may read nothing or the wrong part. `docs/brownfield.md:75` tells adopters "Headings are yours", and `ateam doctor` checks only the protocol's key phrases. Found in SemEngine's v0.4.0 upgrade (C360Studio/semengine#586). The owner chose this fix on 2026-11-11.

## What Changes

- Every skill or contract that tells the agent to read numbered sections gains one fixed sentence: `If your protocol has no such numbered sections, read all of it.` That covers `.agents/contracts/{architect,developer,reviewer}.md:3` and `.agents/skills/{pickup,handoff,preflight}/SKILL.md:10`, plus their copies in `internal/bootstrap/templates.go` (`:351`, `:397`, `:434`, `:482`) and `internal/bootstrap/review.go` (`:19`, `:30`).
- `TestTaskScopedInstructionReads` (`internal/bootstrap/reading_test.go`) also fails when a document that points at numbered sections lacks the sentence.
- In `docs/brownfield.md`, the protocol row says the shipped contracts and skills point at `## 1.` to `## 4.` and *Test-to-spec attribution*. A kept protocol can carry those headings, or a short mapping; otherwise agents fall back to reading the whole protocol.

## Track

`quick`. It modifies one Requirement (*Task-scoped instruction reads*) in one capability. It adds no exported surface, no doctor check, and touches no gate file.

## Non-goals

- No `doctor` warning for missing sections (option B) and no parsing of adopter mappings.
- No renaming of protocol sections. ateam's own layout keeps its context savings.

## Definition of Done

- `TestTaskScopedInstructionReads` passes on the shipped templates and on this repository. `task mutate:check` gives `detection` for removing the sentence from one template.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before the PR is marked ready.
