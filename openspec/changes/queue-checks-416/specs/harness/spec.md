## ADDED Requirements

### Requirement: Queue health checks

`ateam state queue-check` SHALL list, from the forge cache, each `status:blocked` issue whose `Depends on:` issues are all closed or that has none, each `status:needs-decision` issue with a recorded decision, each unplaced open issue, each milestone over `queue.max_milestone_issues` open issues, and each open milestone with none. The pickup line SHALL show `queue=<n>`, or `queue=unavailable` without the data. No command SHALL fail on a finding.

#### Scenario: A blocker that has closed

- **WHEN** an open issue carries `status:blocked` and `Depends on: #5`, and #5 is not in the open set
- **THEN** `ateam state queue-check` reports the issue as blocked on a closed dependency and exits 0

#### Scenario: Unplaced issues are findings

- **WHEN** an open issue has no milestone, or is in a milestone whose `Order:` line omits it
- **THEN** it is reported as unplaced and counted in `queue=`

#### Scenario: Findings never fail a check

- **WHEN** the cache holds findings
- **THEN** `ateam state pickup` and `ateam doctor --check` exit 0, and doctor prints the count
