## MODIFIED Requirements

### Requirement: Not-weaker reviewer by tier and effort band

`ateam review compare --implementer <agent:model:effort>... --reviewer <agent:model:effort>` SHALL apply ADR 0002 rule 2 to `.agents/review-policy.json`: the reviewer passes when its model tier is stronger than every implementer's, or the tiers are equal and its effort band is equal or higher. Compare SHALL exit 0 on a pass, 1 on a finding, and 2 when it cannot decide (bad arguments or an invalid policy).

#### Scenario: Reviewer not weaker

- **WHEN** the reviewer's tier is stronger, or the tiers are equal and the reviewer's band is equal or higher, for every implementer
- **THEN** compare exits 0

#### Scenario: Reviewer weaker

- **WHEN** the reviewer's tier is weaker than any implementer's, or the tiers are equal and its band is lower
- **THEN** compare exits 1 and names the weaker pair

#### Scenario: Unranked or missing effort

- **WHEN** a model or effort is missing from the policy maps (model text matches tier entries ignoring case, with a run of spaces matching one hyphen)
- **THEN** compare passes with an `unranked` note only when both runs of the compared pair are unranked; when one side is ranked it exits 1 naming the unranked side and, for a model, the tier entries its text begins (`GPT-6` suggests `gpt-6-astra`); an unmapped effort matters only between runs of equal tier
- **AND** when a run names no effort, or names a placeholder (`unknown`, `n/a`, `na`, `?`, any case; `none` is a codex effort, not a placeholder), compare exits 1 asking for one

#### Scenario: No policy

- **WHEN** `.agents/review-policy.json` does not exist
- **THEN** compare prints `skipped: no policy` and exits 0
