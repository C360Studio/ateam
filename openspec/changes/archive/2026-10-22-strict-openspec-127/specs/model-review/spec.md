## MODIFIED Requirements

### Requirement: Model tiers and effort bands policy

ATeam SHALL accept a governing policy that declares `model_tiers` (`frontier`, `workhorse`, `light`, each a list of `claude:`, `codex:` or `gemini:` model names) and `effort_bands` (for each of `claude`, `codex` and `gemini`, native effort values listed under `low`, `mid` and `high`) per ADR 0002. Tier entries SHALL be unique across tiers and name a supported provider. Each provider SHALL map at least one effort value, and no value SHALL appear twice for one provider.

#### Scenario: Tiers-only policy

- **WHEN** `.agents/review-policy.json` declares only `version`, `model_tiers`, `effort_bands` and `cross_provider`
- **THEN** the policy parses, `ateam doctor --check` reports no review violations, and `ateam review route` exits non-zero stating that the policy declares model tiers only
- **AND** the policy configures no routes: `review route` and `review check` fail closed naming the missing routes, setup and doctor install and expect no route cards, and the dashboard reports `unconfigured`

#### Scenario: Malformed tiers

- **WHEN** a tier entry lacks a supported provider prefix, appears in two tiers, a provider maps no effort values, an effort value repeats for one provider, or routes, owners, review authors or required checks appear without models
- **THEN** the policy is rejected
