# Change Proposal: `claude-model-ids-209`

Approved-by: Coby (2026-10-28)

## Why

The reviewer contract, `/review-pr` skill, and protocol ask for the "exact model", so a Claude reviewer records its resolved ID, for example `reviewed-by: claude (claude-opus-5-5, high)`. `.agents/review-policy.json` lists only the family aliases `claude:opus`, `claude:sonnet` and `claude:haiku`, and `Policy.tier()` matches tier entries exactly (after lowercasing and replacing spaces with hyphens). The reviewer is therefore unranked and the merge gate rejects a review that should pass (#209, found on SemTeams#363).

## What Changes

- `.agents/review-policy.json`: add current resolved Claude model IDs to the same tier as their family alias (owner ruling, 2026-10-28: current generation only, leaving Opus 4.x and Sonnet 4.x unranked), add `codex:gpt-6-luna` in `light` (owner ruling, 2026-10-28), and move `gemini:gemini-4-argon` from `workhorse` to `frontier` (owner ruling, 2026-10-28).
- ADR 0002 § 1 tier table and policy example: the same entries, plus one sentence saying a tier lists every model spelling a run may record.
- `.agents/contracts/{developer,reviewer}.md` (and `internal/bootstrap/review.go`), `.agents/skills/review-pr/SKILL.md` (and `internal/bootstrap/templates.go`), and `.agents/protocol.md`: ask runs to record the model as `.agents/review-policy.json` lists it, for example `opus`, instead of the "exact model".

## Non-goals

- Family normalization in `internal/review/compare.go` (option 2 in #209); the matching code stays unchanged.
- Model names with the effort attached (`gemini-4-argon-medium`): a recording mistake, not a missing spelling.
- Rewriting existing adopters' policies; `ateam init` seeds no `model_tiers`, so each adopter's policy stays its own.

## Definition of Done

- `ateam review compare --implementer gemini:gemini-4-argon:high --reviewer claude:claude-opus-5-5:high` exits 0 with this repository's policy, `--reviewer claude:opus:high` still exits 0, and `gemini:gemini-4-argon:high` reviews `claude:opus:high` at exit 0.
- `task verify` green on a clean tree; `ateam doctor --check` passes with byte-equal contracts and `review-pr` skill template.
- A test pins the committed policy's ruled Claude IDs, `codex:gpt-6-luna` (`light`), and `gemini:gemini-4-argon` (`frontier`).
- The `Gate files` check passes once `gate-change-approved` is applied to PR #247 by the owner.
