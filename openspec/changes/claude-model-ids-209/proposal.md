# Change Proposal: `claude-model-ids-209`

Approved-by:

## Why

The reviewer contract asks for the "exact model", so a Claude reviewer records its resolved ID, for example `reviewed-by: claude (claude-opus-5-5, high)`. `.agents/review-policy.json` lists only the family aliases `claude:opus`, `claude:sonnet` and `claude:haiku`, and `Policy.tier()` matches tier entries exactly (after lowercasing and replacing spaces with hyphens). The reviewer is therefore unranked and the merge gate rejects a review that should pass (#209, found on SemTeams#363).

## What Changes

- `.agents/review-policy.json`: add current resolved Claude model IDs to the same tier as their family alias (the list is an owner ruling; see design.md, Open Questions).
- ADR 0002 § 1 tier table and policy example: the same entries, plus one sentence saying a tier lists every spelling a run may record.
- `.agents/contracts/{developer,reviewer}.md` and their `ateam review setup` sources in `internal/bootstrap/review.go`: ask runs to record the model as the policy lists it, for example `opus`, instead of the "exact model".

## Non-goals

- Family normalization in `internal/review/compare.go` (option 2 in #209); the matching code stays unchanged.
- Codex and Gemini ID spellings: none has been reported as unranked.
- Rewriting existing adopters' policies; `ateam init` seeds no `model_tiers`, so each adopter's policy stays its own.

## Definition of Done

- `ateam review compare --implementer gemini:gemini-4-argon:high --reviewer claude:claude-opus-5-5:high` exits 0 with this repository's policy, and `--reviewer claude:opus:high` still exits 0.
- `task verify` green on a clean tree; `ateam doctor --check` passes, which means the contracts are byte-equal to their bootstrap sources.
- A test pins at least one resolved ID to its family's tier in the committed policy.
- The `Gate files` check passes using the existing `gate-change-approved` label on #209.
