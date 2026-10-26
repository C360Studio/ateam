## MODIFIED Requirements

### Requirement: OpenSpec queue and stray Hold detection

`ateam state spec-queue` SHALL scan `openspec/changes/*/tasks.md` and report open-task holds (`Hold:`) and caveats (`WONTDO`, `HALT`, `RED`, `BLOCKED`, `OPEN-Q`). `ateam state spec-check` SHALL fail with exit code 1 when any `Hold:` appears below a file's first `## ` heading outside every task block, or exit code 2 when a `tasks.md` is unreadable or invalid UTF-8. Truncated caveat display text SHALL remain valid UTF-8.

#### Scenario: Stray Hold below heading

- **WHEN** an in-flight change's `tasks.md` has `Hold: waiting on #12` in a paragraph outside any `- [ ]` or `- [x]` task block below `## 1. Tasks`
- **THEN** `ateam state spec-check` reports `<path>:<line>:` and exits non-zero

#### Scenario: Multibyte text at the display cut

- **WHEN** a caveat's or hold's text crosses the 104-byte display limit
- **THEN** the displayed text ends before any split multibyte character or partial word (truncating on a word boundary with `…`), is valid UTF-8, and renders a single `Hold:` prefix derived from the captured hold text without re-prefixing the task's leading word or number

#### Scenario: Completed unarchived OpenSpec change

- **WHEN** a change directory remains under `openspec/changes/<id>` with all tasks completed (`N/N`, `N > 0`) and zero caveats
- **THEN** `ateam state spec-queue` prints `done, not archived — move openspec/changes/<id> to openspec/changes/archive/<date>-<id>` instead of `ok`, `BuildPickupSummary` lists it separately under `OpenSpec completed changes (done, not archived)` rather than `OpenSpec in-flight changes`, and `ateam doctor` emits a warning naming the unarchived change

#### Scenario: Proposal awaiting owner approval

- **WHEN** a change's `proposal.md` has an empty `Approved-by:` line and its first open task carries `Hold: OPEN-Q proposal awaits owner approval`, every other task being done
- **THEN** `ateam state spec-queue` reports the change with an empty `approved_by` and flags the hold rather than printing `done, not archived`
- **AND** once the owner fills `Approved-by: <name> (<YYYY-MM-DD>)` and removes the hold, `approved_by` carries that value; an `Approved-by:` line inside a code fence is not the header

#### Scenario: Proposal over one page

- **WHEN** a change's `proposal.md` has 61 or more lines
- **THEN** `ateam state spec-check` prints `warning: openspec/changes/<id>/proposal.md has <n> lines, over the one-page limit of 60` and still exits 0; at 60 lines it prints no warning
