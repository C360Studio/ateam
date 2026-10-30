# Change Proposal: `architect-drift-217`

Approved-by: Coby Leuschke (2026-10-30)

## Why

`ateam init` ships `.agents/contracts/architect.md`, but `reviewFiles` (`internal/bootstrap/review.go:140`) lists only the developer and reviewer contracts. So `ateam review setup` neither writes nor checks the architect contract, and `ateam doctor --check` never reports it as drifted (#217). An adopter's own `architect.md` (SemTeams) can silently drop the base rules the `/design` skill reads.

## What Changes

1. `reviewFiles` adds `.agents/contracts/architect.md` with `architectContract`. The check is the paragraph test the other contracts use (`matchingReviewFile`): the file can add paragraphs, but it must keep every generated paragraph. Setup writes the file when it is missing and refuses before writing when an existing file is missing a paragraph. Doctor reports it as drifted whether or not a policy is installed.
2. `docs/brownfield.md` § Existing specialist contracts says doctor now checks `architect.md` and drops the "until #217" wording.
3. Spec: a scenario added to the model-review Requirement "Explicit setup and portable routing".

## Track

`quick`. One package (`internal/bootstrap`) and one doc. It adds a scenario but no Requirement, adds no exported surface, and touches no boot path, durability behavior, or gate file. Adopter-visible effect: a custom `architect.md` that is missing ateam's paragraphs becomes a doctor violation. The owner chose this on 2026-10-30.

## Non-goals

- An architect route card or provider pointer (`.agents/routes/architect.md`). The policy has no architect route (ADR 0003).
- Migrating any adopter's `architect.md`; the brownfield doc tells them how.

## Definition of Done

- `TestReviewSetupDrift` reports a drifted `architect.md`. `TestConfigureReviewConflictNoWrites` refuses a custom one before writing. `TestConfigureReviewLifecycle` keeps an `architect.md` that has an extra project paragraph.
- `task mutate:check` reports `detection` when the map entry is dropped.
- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
