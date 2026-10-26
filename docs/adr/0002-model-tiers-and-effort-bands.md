# ADR 0002: Model tiers and effort bands for reviewer strength

- **Status:** Accepted (owner, 2026-10-22); § 2 unranked rule amended (owner, #125, 2026-10-22); review record mode added (owner, #130, 2026-10-26; see the addendum)
- **Date:** 2026-10-22
- **Deciders:** repository owner
- **Amends:** [ADR 0001](0001-review-gate-scope.md), decision steps 1 and 2
- **Evidence:** owner rulings on #81 (2026-10-21 and 2026-10-22) and #125 (2026-10-22); the owner's tier table and provider effort-parameter data, posted on #81 and in the Enjoy conversation "Pickup epic 101 v0.3"

## Context

ADR 0001 step 1 asked the owner to declare a model ranking in `.agents/review-policy.json`, and step 2 had merge-check compare reviewer against implementer models through it. Writing that ranking showed four problems:

1. **The policy is expensive to type.** `internal/review.Policy` wants one entry per model with its provider, pairwise `stronger_than` edges to every weaker model, and `routes`, `owners`, `review_authors` and `required_checks` (all required, even though the path that uses them is off). A new model release means editing edges across the list.
2. **Effort doesn't count.** The policy, the reviewer contract ("higher effort … do[es] not prove superiority"), `AGENTS.md` ("effort and provider are not model rank"), `docs/model-review.md` and the model-review spec all treat reasoning effort as unrelated to strength. In practice, the same model at `max` effort is a stronger reviewer than at `low`, and the owner wants that to count.
3. **The ceiling blocks.** Strict superiority means a top-model implementer has no eligible reviewer without an owner exception. ADR 0001 recorded that all three providers here run at their ceilings, so the exception path would be the common path.
4. **Effort vocabularies differ.** Each provider names its effort levels differently, and the valid set differs by model (below). No single word list compares them directly.

Provider effort parameters (owner-supplied, 2026-10-22):

| Provider | Parameter | Values | Notes on current models |
| --- | --- | --- | --- |
| Claude | `output_config.effort` | `low`, `medium`, `high`, `xhigh`, `max` | Fable 5.1, Opus 5.5, Sonnet 5.5 and Haiku 5.5 accept all five |
| Codex / OpenAI | `reasoning.effort`, `model_reasoning_effort` (CLI) | `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` | GPT-6 Astra rejects `none`; GPT-6.1 Sol and GPT-6 Sol reject `none` and `minimal`; GPT-5.4 accepts `none`–`xhigh`. `reasoning.mode` (`standard` / `pro`) is a separate axis |
| Gemini | `thinking_level` | `minimal`, `low`, `medium`, `high` | Gemini 4 Argon, 3.8 Flash and 3.1 Pro accept `low`, `medium`, `high` only |

## Decision

### 1. Three model tiers and three effort bands, typed once in `review-policy.json`

The owner maps each provider model into one of three **tiers**, and each provider's native effort values into one of three **bands**. Both maps live in `.agents/review-policy.json`. PR lines and routes keep native names; the tool does the mapping.

Model tiers (owner ruling, #81, 2026-10-22):

| Tier | Models |
| --- | --- |
| `frontier` | `claude:opus`, `claude:fable`, `codex:gpt-6-astra` |
| `workhorse` | `claude:sonnet`, `codex:gpt-6.1-sol`, `codex:gpt-6-sol`, `gemini:gemini-4-argon` |
| `light` | `claude:haiku`, `codex:gpt-5.4`, `gemini:gemini-3.8-flash`, `gemini:gemini-3.1-pro` |

Effort bands (owner ruling, 2026-10-22: every provider's `high` is the top band):

| Band | Claude | Codex | Gemini |
| --- | --- | --- | --- |
| `low` | `low` | `none`, `minimal`, `low` | `minimal`, `low` |
| `mid` | `medium` | `medium` | `medium` |
| `high` | `high`, `xhigh`, `max` | `high`, `xhigh`, `max` | `high` |

`high` is the top band for every provider, so the bands line up with the one scale all three share (`low` / `medium` / `high`). The owner chose this over a stricter map that kept `high` below `xhigh` and `max`. As a result, a same-tier `max` implementer can be reviewed at `high`.

Codex `reasoning.mode` is not ranked. A `pro` run is compared by its effort only.

Schema sketch:

```json
{
  "version": 1,
  "model_tiers": {
    "frontier": ["claude:opus", "claude:fable", "codex:gpt-6-astra"],
    "workhorse": ["claude:sonnet", "codex:gpt-6.1-sol", "codex:gpt-6-sol", "gemini:gemini-4-argon"],
    "light": ["claude:haiku", "codex:gpt-5.4", "gemini:gemini-3.8-flash", "gemini:gemini-3.1-pro"]
  },
  "effort_bands": {
    "claude": {"low": "low", "medium": "mid", "high": "high", "xhigh": "high", "max": "high"},
    "codex": {"none": "low", "minimal": "low", "low": "low", "medium": "mid", "high": "high", "xhigh": "high", "max": "high"},
    "gemini": {"minimal": "low", "low": "low", "medium": "mid", "high": "high"}
  },
  "cross_provider": false
}
```

A policy containing only `version`, `model_tiers`, `effort_bands` and optionally `cross_provider` is valid for merge-check. `models`, `routes`, `owners`, `review_authors` and `required_checks` become optional. They are needed only by the commands of the dormant path (`review route`, `review check`, setup's route cards), which keep failing closed with a clear message until those fields are present. When the dormant path is enabled, its `stronger_than` relations are derived from tier order instead of typed.

### 2. The rule is "not weaker", by tier then band

A run's rank is its (tier, band) pair. The reviewer passes against an implementer when:

- the reviewer's tier is stronger, or
- the tiers are equal and the reviewer's band is equal or higher.

With several implementer models, the reviewer must pass against every one. Equal (tier, band) passes, so a `frontier` / `high` implementer can always be reviewed by a `frontier` / `high` run. That covers the "higher effort if able" intent without ever blocking at the ceiling.

A model or effort value missing from the maps passes with an `unranked` note only when the other side of the comparison is unranked too. An unranked run against a ranked one is a finding: nothing shows the unranked side is not weaker (owner ruling, #125, 2026-10-22, narrowing #81). The band only breaks a tier tie, so an unmapped effort matters only between runs of equal tier. A new model release therefore blocks a merge against ranked runs until it is added to `model_tiers`, which is a one-line change to the policy.

### 3. PR lines carry model and native effort

Once a policy exists, both PR-body lines must carry the model and its native effort, for example `implemented-by: claude (opus, high)` and `reviewed-by: codex (gpt-6-astra, xhigh)`. A missing effort is a finding, the same as a missing model (#108); a placeholder (`unknown`, `n/a`, `na`, `?`, any case) is a missing effort (#125). An agent that cannot read its own effort names the band it was launched at. `none` is not a placeholder: it is a codex effort. Several implementer runs are comma-separated pairs: `implemented-by: claude (sonnet, medium, opus, high)`. Model text matches the tier names loosely: case and spaces are ignored, so `Gemini 4 Argon` matches `gemini-4-argon` and `GPT-6 Astra` matches `gpt-6-astra`.

Without a policy, the model is still required (#108), effort is optional, and the strength check reports `skipped: no policy`.

### 4. Cross-provider review is optional

Owner ruling (#81, 2026-10-22): cross-service review is optional; "not weaker" is the requirement. Merge-check stops requiring `implemented-by:` and `reviewed-by:` to name different agents by default. The existing `cross_provider` policy flag turns the different-agent rule back on for a repository that wants it.

Independence stays required: the review is a separate run that wrote none of the deliverable. With one shared forge login, that was already an attestation (ADR 0001). The different-agent check was the gate's only mechanical proxy for it, so a same-agent review now rests on the reviewer's written record alone.

Owner ruling (2026-10-26): this repository sets `cross_provider` to `true` in `.agents/review-policy.json`. A same-agent subagent review, spawned by the session that orchestrated the implementation, rested on attestation alone; the different-agent rule restores the mechanical proxy here. The default for other repositories stays optional.

### 5. Instruction text follows

These all change from "effort is not rank" / "a different agent reviews" to "not weaker by tier and band; cross-provider optional":

- the `AGENTS.md` and `.agents/rules-index.md` rows and both bootstrap templates;
- the reviewer and developer contracts (`internal/bootstrap/review.go` → `.agents/contracts/*.md`);
- `.agents/protocol.md`;
- `docs/model-review.md` and its status block;
- the harness and model-review specs.

The dormant path (`review route` / `review check`, strict superiority plus owner exceptions) keeps its current comparison until it is enabled. Its document says so.

## Implementation

1. **#81: the policy.** Parse `model_tiers` and `effort_bands`; make the dormant-path fields optional; have doctor and setup accept a tiers-only policy; commit `.agents/review-policy.json` with the tables above. Tests: a tiers-only policy parses; an unknown band, a model in two tiers, or an unknown provider prefix is rejected.
2. **#82: the gate.** Add `ateam review compare --implementer <agent:model:effort>... --reviewer <agent:model:effort>`, implementing rule 2 over the parsed policy. merge-check calls it, requires effort when a policy exists, and drops the different-agent check unless `cross_provider` is true. Update the instruction text in item 5 in the same PR. Tests cover stronger tier, equal tier with a lower band (fail), equal tier and band (pass), a weaker tier (fail), unranked (pass with note), several implementers, and `cross_provider` on and off. Record mutation detection on the band comparison and the tier comparison.

The two PRs are code PRs and need a review that is not weaker, per this ADR's own rule once #82 lands.

## Consequences

**Positive.** The owner types the ranking once, as two small tables. A new model is one word in a tier. Effort counts. The ceiling never blocks. Any agent can review another's work at an equal or stronger (tier, band), which removes the cross-provider bottleneck that tiers alone would have created (Gemini's best model is `workhorse`).

**Negative.**
- Same-agent review loses the gate's only mechanical independence proxy.
- Effort bands compare different providers' scales by owner judgment, not measurement.
- Tiers are coarse: two models in the same tier may differ a lot on a specific task. The owner accepts that trade for simplicity.
- The dormant path and merge-check use different comparisons until the dormant path is enabled.

**Neutral.**
- `internal/bench` passes native effort to each CLI and doesn't rank, so it is unchanged.
- Ponytail modes (#92) are a separate axis from effort.

## Addendum: review record mode (owner ruling on #130, 2026-10-26)

[Ruling](https://github.com/C360Studio/ateam/issues/130#issuecomment-6416383301): the record switch lives in `.agents/review-policy.json` as `"review_record": "pr-comment" | "aip"`, not in a separate file. Absent means `pr-comment`, and generated output stays byte-identical. Policy validation rejects other values.

Under `aip`, the review record is the archived `openspec/changes/archive/<date>-<id>/review-implementation.md` (AIP's REVIEW-002 format) with `Verdict: PASS` and a matching `approval.md`. ateam agents never post a `reviewed-by:` PR comment. merge-check does not learn to read file records: it does not apply under `aip`, because AIP merges `--ff-only` with no PR. Rule 2 still applies. `ateam review compare --reviewer-record <path>` takes the reviewer run from the record's Reviewer field and runs the same tier-and-band comparison. A verdict other than PASS, or a missing or non-matching `approval.md`, is a finding. `docs/model-review.md` § Review record mode has the details.
