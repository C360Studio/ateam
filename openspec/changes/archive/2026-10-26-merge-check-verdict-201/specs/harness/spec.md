## ADDED Requirements

### Requirement: Merge check requires an approving review verdict

The merge check SHALL accept the latest review record only when it carries exactly one `Verdict: APPROVE` line outside fenced code blocks, and SHALL ignore fenced code blocks when it identifies a review record and its designated commit.

#### Scenario: Rejecting record blocks

- **WHEN** the latest review record names the reviewer and one commit and says `Verdict: REQUEST CHANGES`
- **THEN** the merge check reports a blocking finding naming that verdict and exits non-zero

#### Scenario: Missing or repeated verdict blocks

- **WHEN** the latest review record has no unfenced `Verdict:` line, or more than one
- **THEN** the merge check reports a blocking finding naming what it found and exits non-zero

#### Scenario: Later approval supersedes

- **WHEN** a `Verdict: REQUEST CHANGES` record is followed by a record from the same reviewer with `Verdict: APPROVE` naming the head
- **THEN** the merge check accepts the review record

#### Scenario: Fenced text is not a record

- **WHEN** a comment's only `reviewed-by:`, `Reviewed at`, or `Verdict:` line is inside a ``` or ~~~ fence
- **THEN** the merge check does not count that line
