## ADDED Requirements

### Requirement: Diff-scoped attribution

`ateam test-spec --base <ref>` SHALL report a missing citation only for a supported test whose span (attached comments through its last line) contains a line changed between the merge base of `<ref>` and the working tree, or that lies in an untracked file. Every other diagnostic SHALL apply to all discovered tests. Without `--base` every discovered test SHALL be checked.

#### Scenario: Legacy tests outside the diff

- **WHEN** a repository has uncited tests at `<ref>` and the working tree adds one cited test
- **THEN** `ateam test-spec --base <ref>` exits 0

#### Scenario: New or edited uncited test

- **WHEN** the working tree adds an uncited test, adds an untracked file containing one, or edits a line inside an uncited test that exists at `<ref>`
- **THEN** `ateam test-spec --base <ref>` exits 1 and names each such test

#### Scenario: Broken citation outside the diff

- **WHEN** an unchanged test cites a requirement heading that no longer resolves
- **THEN** `ateam test-spec --base <ref>` exits 1 and names that citation

#### Scenario: Unresolvable base

- **WHEN** `<ref>` is unknown, the root is not a git work tree, or the clone is too shallow to hold the merge base of `<ref>` and `HEAD`
- **THEN** `ateam test-spec --base <ref>` exits 2 naming the ref and how to fetch it, and reports no diagnostics, never falling back to a full or empty scope
