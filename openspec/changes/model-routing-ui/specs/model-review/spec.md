## ADDED Requirements

### Requirement: Read-only dashboard model routing

The authenticated dashboard API SHALL expose configured developer and reviewer model IDs, providers and reasoning efforts separately from model strength, Ponytail discipline mode, runtime identity and PR approval. The UI SHALL render eligible, blocked, unconfigured, invalid and unavailable states with textual reasons. Blocked routes SHALL retain configured identities; missing, unreadable or malformed policy SHALL never imply eligibility. Reads SHALL be bounded to 1 MiB and confined to regular policy files within the selected repository. Routing reads SHALL NOT mutate policy or launch models. Responses SHALL prohibit caching and re-read policy on refresh.

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
