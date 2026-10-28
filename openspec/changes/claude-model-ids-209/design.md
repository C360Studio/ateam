# Design: `claude-model-ids-209`

## Context

- `.agents/review-policy.json:4-6` lists Claude models only as `claude:opus`, `claude:fable` (frontier), `claude:sonnet` (workhorse) and `claude:haiku` (light).
- `internal/review/compare.go:116` `loose()` lowercases a name and turns runs of spaces into one hyphen; `compare.go:118-127` `tier()` returns a rank only when `loose(entry) == provider:loose(model)`, an exact match.
- `internal/review/compare.go:84-95` `modelsLike()` suggests tier entries that start with what the run recorded. `claude-opus-5-5` starts no entry, so the error offers no suggestion.
- `.agents/contracts/reviewer.md:5` says "name the exact model and effort"; `.agents/contracts/developer.md:3` says "exact model". Their `ateam review setup` sources are `internal/bootstrap/review.go:19,32`, and the copies must stay byte-equal (`review.go:230-239`, which `ateam doctor` checks).
- `docs/adr/0002-model-tiers-and-effort-bands.md:36-38` (table) and `:58-60` (example) repeat the policy; `:86` describes matching as loose on case and spaces only.
- `ateam init` writes no `model_tiers`. `review-policy.json` is copied from `--policy FILE` by `ConfigureReview` (`internal/bootstrap/review.go:57`), so the issue's mention of a seeded template does not apply.
- Reproduced at `ad27487`: `ateam review compare --implementer gemini:gemini-4-argon:high --reviewer claude:claude-opus-5-5:high` reports `unranked`.

## Goals / Non-Goals

Goals: a Claude run that records its resolved model ID ranks in the same tier as its family alias, and the contracts stop asking for a spelling the policy cannot rank.

Non-goals: any change to the matching code; Codex and Gemini spellings; adopters' policies.

## Decisions

### D1. Where the ID-to-tier mapping lives (owner ruling on #209: option 1)

| Option | Cost | Blast radius | Forecloses |
| --- | --- | --- | --- |
| **A. List the resolved IDs in the policy (chosen)** | A few JSON entries; no code. | The policy file (a gate file) and the ADR table. | Nothing. Each new Claude ID needs a policy edit with the owner's `gate-change-approved` label, or it ranks as unranked. |
| B. Treat `claude-<family>-*` as the family in `tier()` behind a policy setting | New parser branch, tests and mutation proof in `internal/review/`, which is gate code. | Every adopter that turns the setting on. | Ranking two versions in the same family differently (for example, a weaker old Opus). |

Option A keeps ADR 0002's rule that tiers are typed once and matched literally. The cost is the upkeep: a new model ID fails loudly as `unranked`, a gate miss rather than a false pass, so the failure is safe.

**Adopter seam.** An adopting repository owns its policy and gets none of these entries. If it does nothing, a Claude reviewer that records a full ID stays unranked there, which fails closed. The contract change (D2) reaches adopters through `ateam review setup`, and records `opus` by default, which their alias-only policies already rank.

### D2. What the contracts ask a run to record

| Option | Cost | Forecloses |
| --- | --- | --- |
| **A. "the model as `.agents/review-policy.json` lists it" (chosen)** | Two sentences, in each contract and its bootstrap source. | Nothing: a run can still record a full ID once the policy lists it. |
| B. Keep "exact model" | None. | Adopters with alias-only policies keep failing as in #209. |

Option A changes adopter-visible text. `ateam doctor` reports `missing or drifted review instructions` (`internal/bootstrap/review.go:196`) in an adopter whose contracts match the old text, until they rerun `ateam review setup`.

## Risks / Trade-offs

- **Self-reported identity.** The ID a run records is an attestation either way; listing IDs does not make it more trustworthy.
- **Adopter drift warning.** D2 makes adopters' existing contracts drift. The release notes name `ateam review setup` as the fix.
- **List upkeep.** Each new model ID needs a gate-labelled PR, and ranks as unranked until it lands.

## Migration Plan

No data migration. When this repository's policy changes, PRs with alias records keep passing and full-ID records start passing.

## Open Questions

1. **OWNER RULING — which IDs, in which tier.** Proposed: frontier `claude:claude-opus-5-5`, `claude:claude-fable-5-1`, `claude:claude-opus-4-7`, `claude:claude-opus-4-6`; workhorse `claude:claude-sonnet-5-5`, `claude:claude-sonnet-4-6`; light `claude:claude-haiku-4-5`, `claude:claude-haiku-4-5-20251001`. Should the older Opus 4.x IDs rank as frontier, rank lower, or be left out (unranked)?
