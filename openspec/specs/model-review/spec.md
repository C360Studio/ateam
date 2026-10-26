# Model review Specification

## Purpose

Define explicit project model routing, stronger independent review, scoped owner exceptions and current evidence validation across GitHub and Gitea.

## Requirements

### Requirement: Stronger independent reviewer

On the dormant `review route` / `review check` path, ATeam SHALL require a reviewer run that contributed no implementation and whose actual model is explicitly approved as stronger than every implementation model in the governing project policy. Reasoning effort SHALL be recorded separately and SHALL NOT substitute for model superiority. Same-provider review SHALL be permitted by default; configured cross-provider review SHALL be an additional constraint.

#### Scenario: Stronger model on the same provider

- **WHEN** an independent reviewer uses a model approved as stronger than every implementation model and cross-provider review is not required
- **THEN** the model/provider eligibility checks pass without asserting that the code review itself passed

#### Scenario: Equal model with higher reasoning effort

- **WHEN** the reviewer uses the developer's model with higher reasoning effort and no applicable owner exception exists
- **THEN** approval is blocked

#### Scenario: Different provider without an approved comparison

- **WHEN** a reviewer uses a different provider but policy does not establish superiority over every implementation model
- **THEN** model eligibility is not established and approval is blocked

#### Scenario: Unknown or inherited model identity

- **WHEN** an actual implementation or reviewer model is unknown or inheritance has not been resolved
- **THEN** ATeam reports the missing identity and blocks approval

### Requirement: Explicit owner exception at the model ceiling

When no eligible stronger reviewer is available, ATeam SHALL block approval unless the owner explicitly grants a superiority exception scoped to the reviewed commit, reviewer model and governing policy revision, with owner identity and reason. It SHALL NOT silently reduce developer capability or substitute equal-tier review. An exception SHALL NOT waive independence, other configured constraints, required checks, current evidence or the review verdict.

#### Scenario: Highest approved model without an exception

- **WHEN** implementation used the highest approved model and there is no applicable explicit owner exception
- **THEN** approval remains blocked and the reason identifies the missing eligible reviewer

#### Scenario: New commit after an exception

- **WHEN** the head commit changes after the owner grants an exception
- **THEN** the previous exception does not authorize approval of the new commit

### Requirement: Current and independently attributed review evidence

ATeam SHALL bind review evidence to repository, base, head and policy digest, all implementation runs, and an independent reviewer. Only configured review authors may submit approving evidence. Required checks SHALL have reviewer-executed passing evidence at the current head; reported results are distinct. Changed head or governing policy SHALL invalidate approval. All files, including instructions and rename paths, require review without a Markdown exemption.

#### Scenario: Stale approving record

- **WHEN** an approving review record names a different head or policy revision
- **THEN** approval is blocked pending current evidence

#### Scenario: Model eligible but review requests changes

- **WHEN** the reviewer model is eligible but the review verdict requests changes or required evidence is missing
- **THEN** approval is blocked

#### Scenario: Markdown role contract changes

- **WHEN** a role contract changes or is renamed into an ordinary Markdown path
- **THEN** it remains review-sensitive and cannot bypass the substantive review gate

### Requirement: Portable enforcement and honest trust boundaries

ATeam SHALL validate GitHub/Gitea evidence equivalently, preserve custom setup, diagnose drift, retain two MCP tools and keep network/model launches out of hooks. Model/session/check claims are attestations, not runtime proof. Missing evidence SHALL block approval. Governing policy and owner authority SHALL come from the verified repository default branch, never a candidate-chosen target or candidate policy edit.

#### Scenario: Missing forge evidence

- **WHEN** either forge cannot provide complete review evidence
- **THEN** validation reports the unavailable evidence and does not grant approval

#### Scenario: Existing custom adapter conflicts with policy

- **WHEN** doctor detects that a custom adapter disagrees with the governing model policy
- **THEN** it reports the conflict without silently overwriting the configuration

#### Scenario: Candidate branch weakens its own policy

- **WHEN** a branch changes reviewer policy or exception authority without an authorized policy transition
- **THEN** that change alone does not make the branch eligible for approval

#### Scenario: Candidate targets a permissive branch

- **WHEN** a PR targets a branch other than the verified repository default branch, or its base repository or revision does not match that branch
- **THEN** the gate refuses to treat that target's policy as trusted authority

### Requirement: Explicit setup and portable routing

ATeam SHALL install owner-approved model policy and portable role instructions only through explicit setup. Setup SHALL preserve custom files and refuse conflicts before writing. Routing SHALL select configured models without launching agents, changing native model settings or claiming runtime attestation. Missing policy SHALL block review commands while preserving existing init/doctor compatibility.

#### Scenario: No project-approved model ranking

- **WHEN** no review policy has been installed
- **THEN** route/check refuse approval without inventing model comparisons

#### Scenario: Configured role cards drift

- **WHEN** generated role instructions no longer match the policy
- **THEN** doctor reports drift without overwriting custom configuration

#### Scenario: Reserved pointer belongs to an inactive provider

- **WHEN** a reserved role pointer exists for a provider not selected by the policy, including a dangling symbolic link
- **THEN** setup refuses before writing and doctor reports the exact path without deleting it; unrelated linked provider directories without a role pointer remain compatible

#### Scenario: Canonical contracts without configured policy

- **WHEN** no review policy is installed and a canonical contract is missing or drifted, or reserved route cards or provider pointers remain
- **THEN** doctor reports each invalid artifact while a freshly initialized repository remains valid

#### Scenario: Known edited owner exception

- **WHEN** a comment lacks valid creation/update timestamps, has unequal timestamps, or changes while validation runs
- **THEN** the gate rejects it and requires a fresh comment, without claiming that timestamp equality proves body authorship

### Requirement: Read-only dashboard model routing

The authenticated dashboard API SHALL expose configured developer and reviewer model IDs, providers and reasoning efforts separately from model strength, Ponytail discipline mode, runtime identity and PR approval. The UI SHALL render eligible, blocked, unconfigured, invalid and unavailable states with textual reasons.

#### Scenario: Safe routing reads

- **WHEN** routing is read or refreshed
- **THEN** Blocked routes SHALL retain configured identities; missing, unreadable or malformed policy SHALL never imply eligibility. Reads SHALL be bounded to 1 MiB and confined to regular policy files within the selected repository. Routing reads SHALL NOT mutate policy or launch models. Responses SHALL prohibit caching and re-read policy on refresh.

#### Scenario: Configured eligible and blocked routes

- **WHEN** the selected repository contains a valid review policy
- **THEN** API and UI display exact developer/reviewer model, provider and effort, with eligibility determined by the existing policy validator and a reason when blocked

#### Scenario: Absent or unusable policy

- **WHEN** policy is absent, malformed, oversized, unreadable or not a regular confined file
- **THEN** the routing summary reports unconfigured, invalid or unavailable as appropriate without failing the rest of the dashboard or presenting routes as eligible

#### Scenario: Authenticated fresh display

- **WHEN** a client requests routing or refreshes the dashboard after policy changes
- **THEN** existing read authentication applies, mutation methods are rejected, and the latest non-cacheable summary is returned without changing the policy

#### Scenario: Honest rendering

- **WHEN** the panel receives routing metadata or no metadata from an older server
- **THEN** it displays escaped exact identities and arbitrary effort strings, or unavailable for absent metadata, and explains that configuration is neither runtime identity nor PR approval and effort is separate from strength and Ponytail mode

### Requirement: Model tiers and effort bands policy

ATeam SHALL accept a governing policy that declares `model_tiers` (`frontier`, `workhorse`, `light`, each a list of `claude:`, `codex:` or `gemini:` model names) and `effort_bands` (for each of `claude`, `codex` and `gemini`, native effort values listed under `low`, `mid` and `high`) per ADR 0002. Tier entries SHALL be unique across tiers and name a supported provider. Each provider SHALL map at least one effort value, and no value SHALL appear twice for one provider.

#### Scenario: Tiers-only policy

- **WHEN** `.agents/review-policy.json` declares only `version`, `model_tiers`, `effort_bands` and `cross_provider`
- **THEN** the policy parses, `ateam doctor --check` reports no review violations, and `ateam review route` exits non-zero stating that the policy declares model tiers only
- **AND** the policy configures no routes: `review route` and `review check` fail closed naming the missing routes, setup and doctor install and expect no route cards, and the dashboard reports `unconfigured`

#### Scenario: Malformed tiers

- **WHEN** a tier entry lacks a supported provider prefix, appears in two tiers, a provider maps no effort values, an effort value repeats for one provider, or routes, owners, review authors or required checks appear without models
- **THEN** the policy is rejected

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

### Requirement: Review record mode

`.agents/review-policy.json` SHALL accept an optional `review_record` of `pr-comment` (the default when absent) or `aip` and reject other values. Under `pr-comment`, `ateam init` output SHALL be unchanged. Under `aip`, init, review setup and doctor SHALL name the archived `review-implementation.md` with `Verdict: PASS` and a matching `approval.md` as the review record, with no `reviewed-by:` PR comment and no merge-check.

#### Scenario: Default record mode is unchanged

- **WHEN** the policy has no `review_record`, or sets `pr-comment`, and `ateam init` runs
- **THEN** every generated file is byte-identical to the output with no policy, and `.agents/protocol.md` still asks for a PR comment starting `reviewed-by: <agent> (<model>, <effort>)`

#### Scenario: AIP record mode

- **WHEN** the policy sets `review_record: aip` and `ateam init` or `ateam review setup` runs
- **THEN** the protocol, both rule rows and the reviewer contract name the archived `review-implementation.md` with `Verdict: PASS` and a matching `approval.md`, say ateam agents never post a `reviewed-by:` PR comment and that `task merge:check` does not apply, no longer mention `Reviewed at`, the protocol's Claim and Close and the Close rule rows name no draft PR, `gh pr` command or squash merge but the project's own `--ff-only` merge, the `review-pr` skill checks a record with `ateam review compare --reviewer-record` instead of posting a comment, every other generated file is unchanged, and `ateam doctor --check` passes

#### Scenario: Unknown record mode

- **WHEN** the policy sets `review_record` to any value other than `pr-comment` or `aip`
- **THEN** policy validation rejects it and `ateam init` fails without writing a file

### Requirement: Reviewer run from a review record

`ateam review compare --reviewer-record <path>` SHALL take the reviewer run from the Reviewer field (`<agent> (<model>, <effort>)`) of a REVIEW-002 implementation record and apply the tier-and-band comparison. A record without exactly one Verdict section, `Stage: implementation`, `Verdict: PASS` and parseable Reviewer, or without a sibling `approval.md` row `implementation` holding the record's SHA-256, SHALL be a finding (exit 1). merge-check SHALL NOT read file records.

#### Scenario: Reviewer run from an approved record

- **WHEN** `ateam review compare --implementer <run>... --reviewer-record <path>` names a record with `Verdict: PASS`, a Reviewer starting `<agent> (<model>, <effort>)`, and a sibling `approval.md` whose `implementation` row holds the record's SHA-256
- **THEN** compare applies the tier-and-band comparison to that reviewer run and exits 0 when it is not weaker, 1 when it is weaker
- **AND** a `## Verdict` example inside a ``` or ~~~ code fence elsewhere in the record does not count as a second Verdict section

#### Scenario: Rejected record

- **WHEN** the record's verdict is not `PASS`, `approval.md` is missing or holds no `implementation` row with the record's digest, the Reviewer value names no `claude`, `codex` or `gemini` run, or the Verdict section or one of its fields is missing or repeated (a blank duplicate counts), or the only Verdict section or approval row is inside a ``` or ~~~ code fence
- **THEN** compare prints each finding and exits 1; an unreadable record exits 2, and giving both `--reviewer` and `--reviewer-record` exits 2
