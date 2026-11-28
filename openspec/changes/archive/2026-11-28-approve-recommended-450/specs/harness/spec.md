## MODIFIED Requirements

### Requirement: Opt-in design track

A non-trivial change SHALL declare the `quick` or `design` track at Clarify. `ateam state spec-check` SHALL warn, without failing, when a change's `design.md` exceeds 120 lines. `ateam init` SHALL write the `design` canonical skill with its Claude and Gemini adapters, and `ateam doctor --check` SHALL fail when the canonical `design` skill is missing. `ateam init` SHALL ship Clarify and design guidance in which approving a proposal accepts the recommended option of each decision.

#### Scenario: Design over the length limit

- **WHEN** a change's `design.md` has 121 or more lines
- **THEN** `ateam state spec-check` prints a warning naming `openspec/changes/<id>/design.md`, its line count, and the limit of 120, and still exits 0; at 120 lines, or with no `design.md`, it prints no such warning

#### Scenario: Design skill parity

- **WHEN** `ateam init` runs on an empty directory
- **THEN** it writes `.agents/skills/design/SKILL.md`, `.claude/skills/design/SKILL.md`, and `.gemini/commands/design.toml`
- **AND** `ateam doctor --check` fails once `.agents/skills/design/SKILL.md` is removed

#### Scenario: Init ships the design rules

- **WHEN** `ateam init` runs where no `openspec/config.yaml` exists
- **THEN** it writes `schema: spec-driven` and a `rules: design:` block identical to ateam's own `openspec/config.yaml`, so `openspec instructions design` lists them
- **AND** an existing `openspec/config.yaml` stays byte-for-byte unchanged

#### Scenario: Init ships approval of the recommended options

- **WHEN** `ateam init` runs on an empty directory
- **THEN** the written `.agents/protocol.md`, `.agents/skills/design/SKILL.md` and `openspec/config.yaml` each say that every open question names a recommended option and that a proposal's decisions section opens with "Approving accepts the recommended option of each decision unless you name another."
- **AND** the protocol says a different pick is written into the document and approved at its new digest
