## MODIFIED Requirements

### Requirement: Optional coordinator role

`ateam init` SHALL write a coordinator contract and a canonical `coordinate` skill. The coordinator SHALL route milestone work from `ateam state next` through developer and reviewer runs, and SHALL stop at every owner gate. It SHALL never fill `Approved-by:`, apply `gate-change-approved`, write `reviewed-by:`, or change labels or milestones.

#### Scenario: Init writes the coordinator files

- **WHEN** `ateam init` runs in an empty directory
- **THEN** it writes `.agents/contracts/coordinator.md`, `.agents/skills/coordinate/SKILL.md`, `.claude/skills/coordinate/SKILL.md` and `.gemini/commands/coordinate.toml`, and `ateam doctor --check` applies the canonical-skill checks to `coordinate`

#### Scenario: Owner gates are batched and counted

- **WHEN** a coordinated pull request reaches an owner gate
- **THEN** the coordinator stops that unit, puts the question in one message with every other pending owner question, and the pull request body's `owner-prompts:` line counts each message that covered it

#### Scenario: Developer handoffs surface what the owner must see

- **WHEN** a developer run hands off or rewrites its pull request body
- **THEN** the shipped developer contract puts owner-owed actions in the handoff Goal, names the track with the reason for `quick`, states a gate-file claim only as the line `scripts/gate-files.sh` printed, and keeps any `owner-prompts:` line unchanged
- **AND** the shipped `coordinate` skill's owner message lists those owed actions and repeats each proposal's track
