## ADDED Requirements

### Requirement: Owner decision after the third rejecting review
`ateam review request <pr>` SHALL exit 2, launching nothing and sending nothing to a reviewer session, when the last sync read three or more review records with `Verdict: REQUEST CHANGES` on the PR and the newest unfenced `review-rounds-continue: <n>` comment line does not name that count. Its stderr SHALL name the line to post and list each rejecting round's reviewed sha and reviewer run.

#### Scenario: Third rejection stops the next round
- **WHEN** the cache holds three `REQUEST CHANGES` records for the PR and no continue line
- **THEN** the command exits 2, runs no command, prints no tool text, and stderr names `review-rounds-continue: 3`

#### Scenario: Two rejections pass
- **WHEN** the cache holds two `REQUEST CHANGES` records
- **THEN** the request proceeds as before

#### Scenario: The decision lifts one round
- **WHEN** the cache holds three `REQUEST CHANGES` records and a comment line `review-rounds-continue: 3`
- **THEN** the request proceeds
- **AND** after a fourth `REQUEST CHANGES` record with only that line, the command exits 2 naming `review-rounds-continue: 4`

#### Scenario: Same-session re-review stops too
- **WHEN** `--reviewer-session` is given and the stop applies
- **THEN** nothing is sent to the reviewer session and the command exits 2

#### Scenario: Fenced line ignored
- **WHEN** the only continue line sits inside a code fence
- **THEN** it does not lift the stop
