## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md`, fills `Approved-by: Coby (2026-10-28)`, and removes the approval hold in the same edit (#209).

## 2. Implementation

- [ ] 2.1 Add the ruled Claude IDs in their selected tiers, add `codex:gpt-6-luna` in `light`, and move `gemini:gemini-4-argon` from `workhorse` to `frontier` (design.md, Open Questions 1–3) in `.agents/review-policy.json` (#209). Premise: `.agents/review-policy.json:4-6` lists only `claude:opus|fable|sonnet|haiku` and places `gemini:gemini-4-argon` in `workhorse`; `internal/review/compare.go:118-127` matches `loose(entry) == want`. Check: `ateam review compare --implementer gemini:gemini-4-argon:high --reviewer claude:claude-opus-5-5:high` and `--implementer claude:opus:high --reviewer gemini:gemini-4-argon:high` exit 0.
- [ ] 2.2 Add a test that loads the committed policy and asserts each resolved Claude ID ranks with its alias, `codex:gpt-6-luna` ranks in `light`, and `gemini:gemini-4-argon` ranks in `frontier` (#209). Premise: no test reads the repository's own `.agents/review-policy.json` (`git grep review-policy.json -- '*_test.go'` shows only temp-dir writes). Check: non-destructive mutation checks report `detection` when an entry is removed or demoted.
- [ ] 2.3 Update the ADR 0002 § 1 table, policy example, and status line (#209). Premise: `docs/adr/0002-model-tiers-and-effort-bands.md:3,36-38,58-60`. Check: the table and example match the policy entry for entry.
- [ ] 2.4 Contract, `/review-pr` skill, and protocol wording (#209). Premise: `.agents/contracts/reviewer.md:5`, `.agents/contracts/developer.md:3`, `.agents/skills/review-pr/SKILL.md:35`, and `.agents/protocol.md:98` say "exact model"; their bootstrap sources are `internal/bootstrap/review.go:19,32` and `internal/bootstrap/templates.go:510`. Check: `ateam doctor --check` passes and repository files are byte-equal to their bootstrap sources.
- [ ] 2.5 Archive the change before marking the PR ready (#209).
