# Change Proposal: `review-policy-tiers-81`

## Why

ADR 0002 replaced the pairwise model ranking with three model tiers and three effort bands, typed once in `.agents/review-policy.json`. Before this change, the policy schema required `models`, `stronger_than`, `routes`, `owners`, `review_authors` and `required_checks` even though the path that uses them is off (#81).

## What Changes

- `review.Policy` gains `model_tiers` (`frontier` / `workhorse` / `light` lists of `provider:model`) and `effort_bands` (per provider, native values under `low` / `mid` / `high`). Both use fixed fields, so the strict decoder rejects unknown keys and duplicates.
- A tiers-only policy is valid. `HasRoutes()` gates the dormant path: `review route` and `review check` fail closed naming the tiers-only policy, setup and doctor treat it as configuring no route cards, and the dashboard reports `unconfigured`.
- `.agents/review-policy.json` is committed with the owner's tiers and the looser band map (ADR 0002).
- New model-review spec requirement "Model tiers and effort bands policy". `docs/model-review.md` status block updated.
- Deferred to #82: the tier/band comparison, `ateam review compare`, the merge-check changes, and the instruction-text updates.
