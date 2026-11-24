## ADDED Requirements

### Requirement: Proposal decisions section and spec text length

`ateam state spec-check` SHALL fail when an open change has a `design.md` and its `proposal.md` lacks a `## Decisions you are making by approving` section holding a bullet, and SHALL warn, without failing, when the change has no `design.md`. It SHALL warn on a bullet with fewer than 5 words outside code spans and numbers, a delta Requirement body over 500 characters, and a scenario step over 60 words.

#### Scenario: Missing decisions section on the design track

- **WHEN** an open change has `design.md` and its `proposal.md` has no `## Decisions you are making by approving` section, or the section has no bullet
- **THEN** `ateam state spec-check` names the change's `proposal.md` and the heading to add, and exits 1

#### Scenario: Missing decisions section on the quick track

- **WHEN** an open change has no `design.md` and its `proposal.md` has no decisions section
- **THEN** `ateam state spec-check` prints a warning naming the change's `proposal.md` and still exits 0

#### Scenario: Decision bullet without plain words

- **WHEN** a decisions bullet reads ``- `MaxQueue` = 64``
- **THEN** `ateam state spec-check` warns that the bullet needs plain words and still exits 0
- **AND** a bullet with 5 or more words outside code spans and numbers draws no warning

#### Scenario: Long delta Requirement or scenario step

- **WHEN** an ADDED or MODIFIED Requirement body in an open change is 501 characters, or a scenario step has 61 words
- **THEN** `ateam state spec-check` prints a warning naming the spec file and Requirement, and still exits 0
- **AND** at 500 characters and 60 words it prints no such warning
