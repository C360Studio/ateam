## ADDED Requirements

### Requirement: Review blocking bar and round scope
The reviewer contract SHALL define the findings that block approval and require every other finding to be listed as non-blocking. From the second round, the reviewer SHALL first confirm each prior finding closed and then read the commits since its last record, reading the full diff only when those commits touch files outside it or on request.

#### Scenario: Corner already caught
- **WHEN** a finding is refused by an existing later check and the spec makes no load-bearing promise about the earlier refusal
- **THEN** the record lists it as non-blocking and the verdict can be `APPROVE`

#### Scenario: Re-review message
- **WHEN** `ateam review request --reviewer-session` sends the re-review text
- **THEN** the text asks the reviewer to confirm its prior findings closed and review `<record>..<head>`, and does not ask for the full base-to-head diff
