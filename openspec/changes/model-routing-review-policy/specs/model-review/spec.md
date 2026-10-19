## ADDED Requirements

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

ATeam SHALL validate review evidence against the reviewed head and policy revision, including all implementation model identities, the independent reviewer identity/model/provider/effort, findings, verdict, checks performed and unavailable evidence. It SHALL distinguish checks performed by the reviewer from implementer-reported results. Any changed head or governing policy revision SHALL invalidate prior approval initially. Review-sensitive instructions, role contracts/adapters and policy/configuration files SHALL NOT receive an ordinary documentation exemption, including through a rename.

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

ATeam SHALL provide equivalent GitHub/Gitea policy and review-record validation through existing task/forge boundaries. It SHALL preserve custom adapter configuration, diagnose policy drift, retain two MCP tools and keep network/model launches out of lifecycle hooks. It SHALL distinguish declared identity from authenticated runtime evidence and SHALL NOT treat missing or incomplete forge evidence as approval. Governing policy and owner authority SHALL be resolved from a trusted baseline or explicit authorized transition rather than self-authorized candidate-branch edits.

#### Scenario: Missing forge evidence

- **WHEN** either forge cannot provide complete review evidence
- **THEN** validation reports the unavailable evidence and does not grant approval

#### Scenario: Existing custom adapter conflicts with policy

- **WHEN** doctor detects that a custom adapter disagrees with the governing model policy
- **THEN** it reports the conflict without silently overwriting the configuration

#### Scenario: Candidate branch weakens its own policy

- **WHEN** a branch changes reviewer policy or exception authority without an authorized policy transition
- **THEN** that change alone does not make the branch eligible for approval
