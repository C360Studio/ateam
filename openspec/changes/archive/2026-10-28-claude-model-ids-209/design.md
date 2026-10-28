# Design: `claude-model-ids-209`

## Context

- `.agents/review-policy.json:4-6` lists Claude models only as `claude:opus`, `claude:fable` (frontier), `claude:sonnet` (workhorse) and `claude:haiku` (light).
- `internal/review/compare.go:116` `loose()` lowercases a name and turns runs of spaces into one hyphen; `compare.go:118-127` `tier()` returns a rank only when `loose(entry) == provider:loose(model)`, an exact match.
- `internal/review/compare.go:84-95` `modelsLike()` suggests tier entries that start with what the run recorded. `claude-opus-5-5` starts no entry, so the error offers no suggestion.
- `.agents/contracts/reviewer.md:5`, `.agents/contracts/developer.md:3`, `.agents/skills/review-pr/SKILL.md:35`, and `.agents/protocol.md:98` ask runs to name the "exact model". Their bootstrap sources are `internal/bootstrap/review.go:19,32` (seeded by `templates.go:578-579` and `reviewrecord.go:85`) and `internal/bootstrap/templates.go:510`; `matchingReviewFile` (`review.go:230-238`) checks paragraph containment for contracts in `ateam doctor`, while `TestCanonicalReviewContracts` (`review_test.go:148`) checks byte equality in this repository.
- `docs/adr/0002-model-tiers-and-effort-bands.md:36-38` (table) and `:58-60` (example) repeat the policy; `:86` describes matching as loose on case and spaces only.
- `ateam init` writes no `model_tiers`. `review-policy.json` is copied from `--policy FILE` by `ConfigureReview` (`internal/bootstrap/review.go:57`), so the issue's mention of a seeded template does not apply.
- Codex and Gemini spellings, checked 2026-10-28 against local runtime logs. Codex session files (`~/.codex/sessions/**/*.jsonl`, `"model":` field) record `gpt-6-astra`, `gpt-6-sol`, `gpt-6.1-sol` (all listed) and `gpt-6-luna` (not listed; 8 records from 2026-09-28). They also record `codex-auto-review`, which is Codex's internal approval reviewer and never a session's identity. Recent Gemini and Antigravity CLI files under `~/.gemini/` name only `gemini-4-argon`, `gemini-3.8-flash` and `gemini-3.1-pro` (all listed). PR records show `Gemini 4 Argon`, which `loose()` already matches. Records like `gemini-4-argon-medium` and `Gemini 4 Argon (High)` attach the effort to the model name. That is a recording-format mistake, not a spelling, and adding entries would not fix it.
- Reproduced at `ad27487`: `ateam review compare --implementer gemini:gemini-4-argon:high --reviewer claude:claude-opus-5-5:high` reports `unranked`.

## Goals / Non-Goals

Goals: a Claude run that records its resolved model ID ranks in the same tier as its family alias, and the contracts stop asking for a spelling the policy cannot rank.

Non-goals: any change to the matching code; adopters' policies; accepting a model name with the effort attached.

## Decisions

### D1. Where the ID-to-tier mapping lives (owner ruling on #209: option 1)

| Option | Cost | Blast radius | Forecloses |
| --- | --- | --- | --- |
| **A. List the resolved IDs in the policy (chosen)** | A few JSON entries; no code. | The policy file (a gate file) and the ADR table. | Nothing. Each new Claude ID needs a policy edit with the owner's `gate-change-approved` label, or it ranks as unranked. |
| B. Treat `claude-<family>-*` as the family in `tier()` behind a policy setting | New parser branch, tests and mutation proof in `internal/review/`, which is gate code. | Every adopter that turns the setting on. | Ranking two versions in the same family differently (for example, a weaker old Opus). |

Option A keeps ADR 0002's rule that tiers are typed once and matched literally. The cost is the upkeep: a new model ID fails loudly as `unranked`, a gate miss rather than a false pass, so the failure is safe.

**Adopter seam.** An adopting repository owns its policy and gets none of these entries. If it does nothing, a Claude reviewer that records a full ID stays unranked there, which fails closed. The contract and `/review-pr` wording change (D2) reaches new adopters via `ateam init` / `ateam review setup` and existing adopters via `ateam init --force` (`internal/bootstrap/bootstrap.go:89-127`), recording `opus` by default so alias-only policies rank it.

### D2. What the contracts, `/review-pr` skill, and protocol ask a run to record

| Option | Cost | Forecloses |
| --- | --- | --- |
| **A. "the model as `.agents/review-policy.json` lists it" (chosen)** | Small prose edits in both contracts, `review-pr/SKILL.md`, `protocol.md`, and their bootstrap sources. | Nothing: a run can still record a full ID once the policy lists it. |
| B. Keep "exact model" | None. | Adopters with alias-only policies keep failing as in #209. |

Option A changes adopter-visible text. `ateam doctor` reports `missing or drifted review instructions` (`internal/bootstrap/review.go:196`) in an adopter whose contracts still hold the old paragraph. Because `ConfigureReview` (`review.go:85-96`) never overwrites existing files, adopters refresh existing contracts and skills with `ateam init --force` (preserving `review_record: aip` via `reviewrecord.go:68-88`), or reconcile custom additions manually.

## Risks / Trade-offs

- **Self-reported identity.** The ID a run records is an attestation either way; listing IDs does not make it more trustworthy.
- **Adopter drift warning.** D2 makes adopters' existing contracts drift until they run `ateam init --force` (or reconcile custom contract paragraphs).
- **List upkeep.** Each new model ID needs a gate-labelled PR, and ranks as unranked until it lands.

## Migration Plan

No data migration. When this repository's policy changes, PRs with alias records keep passing, full-ID Claude records start passing, and `gemini:gemini-4-argon` ranks in `frontier`.

## Open Questions

1. **Resolved (owner, 2026-10-28): which IDs, in which tier.** frontier `claude:claude-opus-5-5`, `claude:claude-fable-5-1`; workhorse `claude:claude-sonnet-5-5`; light `claude:claude-haiku-4-5`, `claude:claude-haiku-4-5-20251001`. The Opus 4.x and Sonnet 4.x IDs are left out and rank as unranked.
2. **Resolved (owner, 2026-10-28): `codex:gpt-6-luna` belongs in the light tier.** Add that exact runtime ID to the policy and ADR 0002; the other observed Codex and Gemini IDs are already listed.
3. **Resolved (owner, 2026-10-28): `gemini:gemini-4-argon` belongs in the frontier tier.** Move `gemini:gemini-4-argon` from `workhorse` to `frontier` in `.agents/review-policy.json` and ADR 0002.
