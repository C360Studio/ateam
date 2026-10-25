## ADDED Requirements

### Requirement: Adopter rules parity

`ateam doctor` SHALL check an adopter's rule files for agreement with each other, and its protocol for the markers the harness depends on, without requiring ateam's own section titles or protocol headings.

#### Scenario: Adopter rule sections that agree

- **WHEN** `AGENTS.md` and `.agents/rules-index.md` have the same numbered rule sections, with the same titles in the same order and the same row count per section, whatever the titles, their number, or the heading level used in each file (per file, the heading level whose numbered sections own the most table rows, `###` on a tie; other numbered headings are ignored), and whatever the table's header wording, separator alignment (a separator row has at least one `-`; an all-blank row is a row), indentation or spacing after `|`
- **THEN** doctor reports no rules-parity violation

#### Scenario: Adopter rule sections that disagree

- **WHEN** a numbered section's title, position, or row count differs between `AGENTS.md` and `.agents/rules-index.md`
- **THEN** doctor reports a violation naming the section and the file in which it differs

#### Scenario: Adopter rule section without table rows

- **WHEN** a numbered rule section in `AGENTS.md` or `.agents/rules-index.md` has no table rows, even when the other file agrees
- **THEN** doctor reports a violation naming the section and the file

#### Scenario: Protocol markers

- **WHEN** `.agents/protocol.md` lacks any of `Deliverable`, `non-goals`, `Definition of Done`, `Stop boundary`, `ateam state handoff`, or `openspec/changes/archive/`
- **THEN** doctor reports a violation naming the missing obligation and where the upstream template puts it, and its headings are not checked
