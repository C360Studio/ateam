## MODIFIED Requirements

### Requirement: Task-scoped instruction reads

Canonical skills, role contracts and their `ateam init` templates SHALL name the protocol sections their workflow needs instead of requiring a full read of `.agents/protocol.md`, and every section they name SHALL exist in the shipped protocol template.

#### Scenario: Pointers resolve

- **WHEN** a shipped skill or contract template names a protocol section `§ n`
- **THEN** the shipped protocol template has a `## n.` heading

#### Scenario: No unconditional full read

- **WHEN** an agent starts a pickup, handoff, preflight, design or review workflow
- **THEN** its skill and contract direct it to named sections, not to the whole protocol

#### Scenario: A protocol without the numbered sections

- **WHEN** a skill or contract tells the agent to read numbered protocol sections
- **THEN** it also says `If your protocol has no such numbered sections, read all of it.`, so an adopter's own protocol is read whole rather than not at all
