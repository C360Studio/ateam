## MODIFIED Requirements

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

#### Scenario: Architect contract drift

- **WHEN** `.agents/contracts/architect.md` is missing a paragraph of the generated architect contract
- **THEN** `ateam review setup` refuses before writing and doctor reports the path, while an architect contract that adds its own paragraphs to the generated ones remains valid
