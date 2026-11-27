## ADDED Requirements

### Requirement: Recorded decision lines

An issue's recorded decision SHALL be the first matching line of its newest comment that has one, else of its body, else a label starting `decision:`. After trimming whitespace and leading `*` or `_`, a line SHALL match when it starts, ignoring case, with `Decision:` or `Ruling:`, or with `Owner decision` or `Owner ruling` and contains a `YYYY-MM-DD` date. `ateam state queue-check` and the session-start summary SHALL read it.

#### Scenario: A dated owner ruling is recorded

- **WHEN** an issue's newest comment has the line `**Owner ruling (2026-10-30):** keep the cap`
- **THEN** that line is the issue's recorded decision

#### Scenario: An undated owner phrase is not a decision

- **WHEN** a `status:needs-decision` issue's body has the line `Owner decision needed: pick A or B` and no comment has a matching line
- **THEN** the issue has no recorded decision and `ateam state queue-check` does not report it as decided

#### Scenario: The existing prefixes still match

- **WHEN** an issue comment has the line `Decision: use Gitea v1 issues`
- **THEN** that line is the issue's recorded decision
