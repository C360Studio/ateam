## ADDED Requirements

### Requirement: Committed team settings file
`ateam` SHALL read optional team settings from the committed, non-gate file `.agents/ateam.json`. The file SHALL have version 1 and be strictly decoded. Its optional keys are:
- `merge_method`: `squash`, `merge` or `rebase`, default `squash`;
- `default_branch`;
- `commit_statuses`: default `true`;
- `forges`.

A missing file SHALL leave every behaviour and every `ateam init` output byte-identical to a repository without the file. Keys that can weaken a merge gate SHALL NOT be accepted in this file.

#### Scenario: Missing file
- **WHEN** `.agents/ateam.json` does not exist
- **THEN** `ateam init` writes the same bytes as before and does not create the file
- **AND** `ateam doctor` reports nothing about it

#### Scenario: Invalid file
- **WHEN** the file has an unknown key, a version other than 1, an out-of-range value, or trailing data
- **THEN** `ateam doctor --check` fails and names the file and the error
- **AND** `ateam init` returns the error
- **AND** hooks proceed with defaults

#### Scenario: Merge method renders the Close text
- **WHEN** `merge_method` is `merge` or `rebase` and `review_record` is not `aip`
- **THEN** the generated Close rule and protocol Close step name `gh pr merge --merge --auto` or `gh pr merge --rebase --auto`, and the matching Gitea `do` value `merge` or `rebase`
- **AND** `ateam doctor` warns when `.agents/protocol.md` does not name the configured method

#### Scenario: Configured default branch
- **WHEN** `default_branch` is set
- **THEN** only that branch counts as the default branch for worktree labels, prune, handoff freshness, and ahead/behind counts
- **AND** when it is unset, `main` and `master` both count as the default branch, as today

#### Scenario: Commit statuses precedence
- **WHEN** `ATEAM_COMMIT_STATUSES` is `0` or `1`
- **THEN** it overrides `commit_statuses` on that machine
- **AND** otherwise `commit_statuses: false` disables commit statuses, and an absent key leaves them enabled

#### Scenario: Committed forge list
- **WHEN** `forges` is set
- **THEN** it replaces auto-detection and any forge list edited locally
- **AND** the board's forge settings API refuses edits with a message that names the file
- **AND** when it is unset, forge detection and board edits behave as before
