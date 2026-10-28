## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md`, rules on the ID list (design.md, Open Questions), fills `Approved-by: <name> (<YYYY-MM-DD>)` and removes this hold in the same edit (#209). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Add the ruled IDs to `.agents/review-policy.json` (#209). Premise: `.agents/review-policy.json:4-6` lists only `claude:opus|fable|sonnet|haiku`; `internal/review/compare.go:118-127` matches `loose(entry) == want`. Check: `ateam review compare --implementer gemini:gemini-4-argon:high --reviewer claude:claude-opus-5-5:high` exits 0.
- [ ] 2.2 Add a test that loads the committed policy and asserts each resolved ID ranks with its alias (#209). Premise: no test reads the repository's own `.agents/review-policy.json` (`git grep review-policy.json -- '*_test.go'` shows only temp-dir writes). Check: `task mutate:check` reports `detection` when one ID is removed from the policy.
- [ ] 2.3 Update the ADR 0002 § 1 table and policy example (#209). Premise: `docs/adr/0002-model-tiers-and-effort-bands.md:36-38,58-60`. Check: the table matches the policy entry for entry.
- [ ] 2.4 Contract wording (#209). Premise: `.agents/contracts/reviewer.md:5` and `.agents/contracts/developer.md:3` say "exact model"; their sources are `internal/bootstrap/review.go:19,32`. Check: `ateam doctor --check` passes and both copies are byte-equal.
- [ ] 2.5 Archive the change before marking the PR ready (#209).
