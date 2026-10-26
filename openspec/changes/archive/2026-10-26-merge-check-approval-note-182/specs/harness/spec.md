## ADDED Requirements

### Requirement: Merge check notes an unapproved OpenSpec change

On a code pull request, the merge check SHALL print a note for each OpenSpec change directory the pull request touches whose `proposal.md` at the head lacks a filled `Approved-by:` header, and the note SHALL NOT change the exit status.

#### Scenario: Unapproved change on a code pull request

- **WHEN** a code pull request adds files under `openspec/changes/foo/` and `openspec/changes/foo/proposal.md` at its head has `Approved-by:` with no value
- **THEN** the merge check prints `merge-check: note: OpenSpec change foo has no Approved-by; the spec was not human-reviewed` and its exit status is what it would be without the note

#### Scenario: Approved change

- **WHEN** the touched change's `proposal.md` has `Approved-by: Owner (2026-10-26)`
- **THEN** the merge check prints no note for it

#### Scenario: Archived change and unreadable proposal

- **WHEN** the pull request touches `openspec/changes/archive/2026-10-26-foo/` whose `proposal.md` is missing, or the read of a touched `proposal.md` fails
- **THEN** the merge check prints a note naming the change or the path, and a failed read does not make the check unavailable
