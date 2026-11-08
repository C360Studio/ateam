## MODIFIED Requirements

### Requirement: Committed team settings file
`ateam` SHALL read optional team settings (`merge_method`, `default_branch`, `commit_statuses`, `forges`, `ateam_version`) from the committed, non-gate, strictly decoded version-1 file `.agents/ateam.json`. A missing file SHALL change no behaviour and no `ateam init` output. The file SHALL NOT accept keys that can weaken a merge gate.

#### Scenario: Missing file
- **WHEN** `.agents/ateam.json` does not exist
- **THEN** `ateam init` writes the same bytes as before and does not create the file
- **AND** `ateam doctor` reports nothing about it

#### Scenario: Absent keys take defaults
- **WHEN** the file omits a key
- **THEN** `merge_method` is `squash`, `commit_statuses` is `true`, and `default_branch` and `forges` behave as if no file existed

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
- **AND** an empty list is an error that says to omit the key to auto-detect
- **AND** when it is unset, forge detection and board edits behave as before
- **AND** once it is removed, or the file is deleted or invalid, the next refresh restores the board-edited list it replaced, else detects the forge again

### Requirement: Doctor checks a declared ateam pin

Outside the ateam checkout, when `ateam_version` in `.agents/ateam.json` or the deprecated `.agents/ateam-version` declares a pin, `ateam doctor` SHALL report it, preferring the key. It SHALL add a violation when the running binary's release, or the release of the binary the Claude hook command resolves to, differs from it by exact string. It SHALL also add a violation when the pin is not one `v<major>.<minor>.<patch>` release. Without a pin, doctor's output SHALL be unchanged.

#### Scenario: Dev build against a release pin

- **WHEN** `.agents/ateam-version` holds `v0.3.9` and the hook command resolves to a binary reporting `v0.3.9-8-gbd6941f`
- **THEN** doctor reports a violation naming the hook binary's path, its version, and the pin, and exits non-zero

#### Scenario: Both binaries at the pin

- **WHEN** the pin is `v0.3.9` and both the running binary and the hook binary report `v0.3.9`
- **THEN** doctor adds no pin violation

#### Scenario: No hook binary on a CI runner

- **WHEN** the pin is `v0.3.9`, the running binary reports `v0.3.9`, and no hook binary resolves
- **THEN** doctor reports `hook_binary` as absent and adds no pin violation

#### Scenario: No pin declared

- **WHEN** neither `ateam_version` nor `.agents/ateam-version` exists
- **THEN** doctor's output carries no pin field and no pin finding

#### Scenario: Key and file disagree

- **WHEN** `ateam_version` is `v0.4.0` and `.agents/ateam-version` holds `v0.3.9`
- **THEN** doctor checks against `v0.4.0` and warns that the key wins over the file
