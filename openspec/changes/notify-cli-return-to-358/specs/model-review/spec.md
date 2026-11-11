## MODIFIED Requirements

### Requirement: Review notify text
`ateam review notify` SHALL send only the fixed text `review record for PR #<n> landed at <sha>; run ateam state sync, then ateam state pickup`, followed by `; to re-review after fixes run ateam review request --reviewer-session <id> --return-to <return_to> <n>` when the reviewer's own session is known, to the launch spec's `return_to` through the host manifest's `notify` command. It SHALL print the host tool when there is no `notify` and send nothing on `cli`.

#### Scenario: Hostile record body
- **WHEN** the review record comment carries shell text and the different-writer warning line
- **THEN** the message sent is the fixed text and nothing from the comment

#### Scenario: Reviewer session in the text
- **WHEN** the reviewer's session comes from its environment or `--from` and is one line of `[A-Za-z0-9._:-]`
- **THEN** the message ends with the re-review clause naming that session and the `--return-to` it was sent to
- **AND** a session or `--return-to` holding any other character exits 2 and sends nothing

#### Scenario: A return_to on the cli host
- **WHEN** the host resolves to `cli` and `--return-to` names a session
- **THEN** nothing is sent, the command exits 1, stdout carries the message and the target to deliver by hand, and stderr names `ATEAM_HOST`
- **AND** with no `--return-to` on `cli` it exits 0 and prints nothing
