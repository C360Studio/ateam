## MODIFIED Requirements

### Requirement: Re-review to the reviewing session
`ateam review request --reviewer-session <id> <pr>` SHALL send the fixed re-review text to `<id>` through the host manifest's `notify` action only when the cached newest record is `REQUEST CHANGES` or `APPROVE` at a sha other than the PR head and its reviewer run passes the gate comparison against the current implementers; otherwise, or with `--fresh`, it SHALL print the fresh launch spec naming why.

#### Scenario: Same reviewer after changes requested
- **WHEN** the cached record is `REQUEST CHANGES` at an older sha and its run still passes the comparison
- **THEN** the re-review text naming the record sha and the head is sent to the reviewer session and no launch runs

#### Scenario: Same reviewer after an approved PR moved
- **WHEN** the cached record is `APPROVE` at an older sha, as after a catch-up merge of `main` touching the PR's files, and its run still passes the comparison
- **THEN** the re-review text naming `APPROVE`, the record sha and the head is sent to the reviewer session and no launch runs

#### Scenario: Fresh launch instead
- **WHEN** the record has neither verdict, is at the head, is absent, names a run the gate now rejects, or `--fresh` is given
- **THEN** the fresh launch spec is printed with `rereview` naming the reason and nothing is sent to the reviewer session

#### Scenario: Reviewer unreachable
- **WHEN** the manifest's `notify` command fails for the reviewer session
- **THEN** the fresh launch spec is printed with `rereview: unreachable`

### Requirement: Review blocking bar and round scope
The reviewer contract SHALL define the findings that block approval and require every other finding to be listed as non-blocking. From the second round, the reviewer SHALL first confirm each prior finding closed and then read the commits since its last record, reading the full diff only when those commits touch files outside it or on request.

#### Scenario: Corner already caught
- **WHEN** a finding is refused by an existing later check and the spec makes no load-bearing promise about the earlier refusal
- **THEN** the record lists it as non-blocking and the verdict can be `APPROVE`

#### Scenario: Re-review message
- **WHEN** `ateam review request --reviewer-session` sends the re-review text after `REQUEST CHANGES`
- **THEN** the text asks the reviewer to confirm its prior findings closed and review `<record>..<head>`, and does not ask for the full base-to-head diff
- **AND WHEN** it sends the text after `APPROVE`
- **THEN** the text asks the reviewer to review `<record>..<head>` without asking it to confirm findings closed, and does not ask for the full base-to-head diff

