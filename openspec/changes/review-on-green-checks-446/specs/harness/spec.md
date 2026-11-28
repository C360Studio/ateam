## MODIFIED Requirements

### Requirement: Optional coordinator role

`ateam init` SHALL write a coordinator contract and a canonical `coordinate` skill. The coordinator SHALL route milestone work from `ateam state next` through developer and reviewer runs, and SHALL stop at every owner gate. It SHALL never fill `Approved-by:`, apply `gate-change-approved`, write `reviewed-by:`, or change labels or milestones.

#### Scenario: Init writes the coordinator files

- **WHEN** `ateam init` runs in an empty directory
- **THEN** it writes `.agents/contracts/coordinator.md`, `.agents/skills/coordinate/SKILL.md`, `.claude/skills/coordinate/SKILL.md` and `.gemini/commands/coordinate.toml`, and `ateam doctor --check` applies the canonical-skill checks to `coordinate`

#### Scenario: Owner gates are batched and counted

- **WHEN** a coordinated pull request reaches an owner gate
- **THEN** the coordinator stops that unit, puts the question in one message with every other pending owner question, and the pull request body's `owner-prompts:` line counts each message that covered it

#### Scenario: Review is requested only on green required checks

- **WHEN** a developer run hands off a coordinated pull request at ready-for-review
- **THEN** the shipped developer contract allows a `ready` handoff only once every required check at the pull request head other than `Merge check` has passed, and the shipped `coordinate` skill reads those checks before `ateam review request`
- **AND WHEN** one of them failed or is still running, the coordinator messages the developer run instead of requesting review
