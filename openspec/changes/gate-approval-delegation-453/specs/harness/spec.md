## MODIFIED Requirements

### Requirement: Optional coordinator role

`ateam init` SHALL write a coordinator contract and a canonical `coordinate` skill. The coordinator SHALL route milestone work from `ateam state next` through developer and reviewer runs, and SHALL stop at every owner gate. It SHALL never fill `Approved-by:`, apply `gate-change-approved`, write `reviewed-by:`, or change labels or milestones.

#### Scenario: Init writes the coordinator files

- **WHEN** `ateam init` runs in an empty directory
- **THEN** it writes `.agents/contracts/coordinator.md`, `.agents/skills/coordinate/SKILL.md`, `.claude/skills/coordinate/SKILL.md` and `.gemini/commands/coordinate.toml`, and `ateam doctor --check` applies the canonical-skill checks to `coordinate`

#### Scenario: Owner gates are batched and counted

- **WHEN** a coordinated pull request reaches an owner gate
- **THEN** the coordinator stops that unit, puts the question in one message with every other pending owner question, and the pull request body's `owner-prompts:` line counts each message that first asked about it
- **AND WHEN** a later message repeats a gate already asked and still unanswered, that pull request's `owner-prompts:` line is unchanged

#### Scenario: Review is requested only on green required checks

- **WHEN** a developer run hands off a coordinated pull request at ready-for-review
- **THEN** the shipped developer contract allows a `ready` handoff only once every required check at the pull request head other than `Merge check` has passed, and the shipped `coordinate` skill reads those checks before `ateam review request`
- **AND** on a pull request touching a gate file, `Gate files` is excused as well while it waits for the owner's approval, which the developer asks for last; no other check is excused, and the pull request still merges only once `Gate files` passes at its head
- **AND WHEN** any other of them failed or is still running, the coordinator messages the developer run instead of requesting review

#### Scenario: Developer handoffs surface what the owner must see

- **WHEN** a developer run hands off or rewrites its pull request body
- **THEN** the shipped developer contract puts owner-owed actions in the handoff Goal, names the track with the reason for `quick`, states a gate-file claim only as the line `scripts/gate-files.sh` printed, and keeps any `owner-prompts:` line unchanged
- **AND** the shipped `coordinate` skill's owner message lists those owed actions and repeats each proposal's track

#### Scenario: Wake-up reasons cover every developer signal

- **WHEN** the shipped `coordinate` skill starts a developer run
- **THEN** its launch prompt sends the wake-up line once the draft PR is open and after every handoff, with the reason `claimed`, `ready`, `blocked`, `question`, `holding` or `closed`, and the shipped developer contract sends a design-track proposal's open questions as `question` naming the `design.md` section
- **AND WHEN** a wake-up arrives for a handoff the coordinator has already read, the skill has it acknowledge the wake-up and do nothing else

#### Scenario: Proposal approvals offer only working routes and are not re-asked

- **WHEN** the shipped `coordinate` skill asks the owner to approve a proposal
- **THEN** the message gives the developer run's conversation link before the pull request and offers approval through the coordinator, in that developer conversation, or by the owner's own commit, leaving out a route the developer reported its host refused
- **AND** before sending, the skill syncs and reads the branch and pull request, and drops a gate already answered
- **AND** the shipped developer contract writes `Approved-by:` only on the owner's own words, quoting them in the commit, and hands off as `blocked` when its host refuses the edit
