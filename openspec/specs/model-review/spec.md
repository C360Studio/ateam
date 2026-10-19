# Model review Specification

## Purpose

Define explicit project model routing, stronger independent review, scoped owner exceptions and current evidence validation across GitHub and Gitea.

## Requirements

### Requirement: Stronger independent reviewer

ATeam SHALL require a reviewer run that contributed no implementation and whose actual model is explicitly approved as stronger than every implementation model in the governing project policy. Reasoning effort SHALL be recorded separately and SHALL NOT substitute for model superiority. Same-provider review SHALL be permitted by default; configured cross-provider review SHALL be an additional constraint.

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

The authenticated dashboard API SHALL expose configured developer and reviewer model IDs, providers and reasoning efforts separately from model strength, Ponytail discipline mode, runtime identity and PR approval. The UI SHALL render eligible, blocked, unconfigured, invalid and unavailable states with textual reasons. Blocked routes SHALL retain configured identities; missing, unreadable or malformed policy SHALL never imply eligibility. Reads SHALL be bounded to 1 MiB and confined to regular policy files within the served repository. Routing reads SHALL NOT mutate policy or launch models. Responses SHALL prohibit caching and re-read policy on refresh.

#### Scenario: Configured eligible and blocked routes

- **WHEN** the served checkout contains a valid review policy
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
