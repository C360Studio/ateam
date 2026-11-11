## ADDED Requirements

### Requirement: Re-review to the reviewing session
`ateam review request --reviewer-session <id> <pr>` SHALL send the fixed re-review text to `<id>` through the host manifest's `notify` action only when the cached newest record is `REQUEST CHANGES` at a sha other than the PR head and its reviewer run passes the gate comparison against the current implementers; otherwise, or with `--fresh`, it SHALL print the fresh launch spec naming why.

#### Scenario: Same reviewer after changes requested
- **WHEN** the cached record is `REQUEST CHANGES` at an older sha and its run still passes the comparison
- **THEN** the re-review text naming the record sha and the head is sent to the reviewer session and no launch runs

#### Scenario: Fresh launch instead
- **WHEN** the record is `APPROVE`, is at the head, is absent, names a run the gate now rejects, or `--fresh` is given
- **THEN** the fresh launch spec is printed with `rereview` naming the reason and nothing is sent to the reviewer session

#### Scenario: Reviewer unreachable
- **WHEN** the manifest's `notify` command fails for the reviewer session
- **THEN** the fresh launch spec is printed with `rereview: unreachable`

## MODIFIED Requirements

### Requirement: Review notify text
`ateam review notify` SHALL send only the fixed text `review record for PR #<n> landed at <sha>; run ateam state sync, then ateam state pickup`, followed by `; to re-review after fixes run ateam review request --reviewer-session <id> <n>` when the reviewer's own session is known, to the launch spec's `return_to` through the host manifest's `notify` command. It SHALL print the host tool when there is no `notify` and send nothing on `cli`.

#### Scenario: Hostile record body
- **WHEN** the review record comment carries shell text and the different-writer warning line
- **THEN** the message sent is the fixed text and nothing from the comment

#### Scenario: Reviewer session in the text
- **WHEN** the reviewer's session comes from its environment or `--from` and is one line of `[A-Za-z0-9._:-]`
- **THEN** the message ends with the re-review clause naming that session
- **AND** a session holding any other character exits 2 and sends nothing
