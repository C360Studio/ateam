## MODIFIED Requirements

### Requirement: OpenSpec queue and stray Hold detection

`ateam state spec-queue` SHALL scan `openspec/changes/*/tasks.md` and report open-task holds (`Hold:`) and caveats (`WONTDO`, `HALT`, `RED`, `BLOCKED`, `OPEN-Q`). `ateam state spec-check` SHALL fail with exit code 1 when any `Hold:` appears below a file's first `## ` heading outside every task block, or when an in-flight change's filled `Approved-by:` carries a `sha256:` digest that no longer matches the digest of its `proposal.md` (without the `Approved-by:` line) and `design.md`, or exit code 2 when a `tasks.md` is unreadable or invalid UTF-8. Truncated caveat display text SHALL remain valid UTF-8.

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
- **THEN** `ateam state spec-queue` reports the change with an empty `approved_by`, flags the hold rather than printing `done, not archived`, and prints the current digest `sha256:<12 hex>` of `proposal.md` (without the `Approved-by:` line) and `design.md`
- **AND** once the owner fills `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removes the hold, `approved_by` carries that value and the change is labelled `approved`; an `Approved-by:` line inside a code fence is not the header

#### Scenario: Stale approval

- **WHEN** an in-flight change's `proposal.md` or `design.md` changes after its `Approved-by:` line was filled with a digest, or a `design.md` is added
- **THEN** `ateam state spec-check` exits 1 naming the changed file, the last commit that touched it or `uncommitted`, and the current digest, and `ateam state spec-queue` labels the change `stale approval`
- **AND** filling the line with the current digest, removing the hold, and moving the change to `openspec/changes/archive/` leave the digest unchanged

#### Scenario: Approval without a digest

- **WHEN** an in-flight change's `Approved-by:` is filled without a `sha256:` token
- **THEN** `ateam state spec-check` prints a warning labelling the change `approval unbound` and still exits 0, and a change under `archive/` is never checked

#### Scenario: Proposal over one page

- **WHEN** a change's `proposal.md` has 61 or more lines
- **THEN** `ateam state spec-check` prints `warning: openspec/changes/<id>/proposal.md has <n> lines, over the one-page limit of 60` and still exits 0; at 60 lines it prints no warning

#### Scenario: Strict validation names what passed

- **WHEN** `ateam state spec-check` finds `node_modules/.bin/openspec` or `openspec` on `PATH` and `openspec validate --all --strict` exits 0
- **THEN** it prints `openspec: strict ok (<resolved path>, <version>, <n> changes)`, where `<version>` is the first line of `openspec --version` or `version unknown`, and `<n>` counts the in-flight changes
- **AND** when no validator is found it prints the skip notice and no `strict ok` line, and when validation fails it exits 1 with the validator's output and no `strict ok` line

### Requirement: Informational commit statuses on the PR head
`ateam state sync`, per open pull request whose branch has a local worktree, and a posting handoff (CLI or MCP), for its branch's open pull request, SHALL set the `ateam/proposal`, `ateam/handoff`, and `ateam/review-record` commit statuses on the PR head sha with the write token, never from a hook and never as a required check. A status SHALL be posted only when its state or description differs from the latest with its context on that sha. A failed write SHALL NOT change an exit code.

#### Scenario: Proposal approved or awaiting approval
- **WHEN** the PR touches an OpenSpec change whose `proposal.md` at the head has `Approved-by:` filled and no open `OPEN-Q` hold
- **THEN** `ateam/proposal` is `success` naming the approver without the digest token
- **AND** with `Approved-by:` empty or the hold open it is `pending` `awaiting owner approval (OPEN-Q)`

#### Scenario: Stale approval at the head
- **WHEN** a touched change's `Approved-by:` digest does not match its `proposal.md` and `design.md` at the PR head
- **THEN** `ateam/proposal` is `pending` `approval stale: <id>`

#### Scenario: No change touched
- **WHEN** the PR touches no directory under `openspec/changes/`
- **THEN** no `ateam/proposal` status is posted

#### Scenario: Handoff fresh or stale
- **WHEN** the newest handoff on the branch has a checkpoint equal to the PR head and a clean tree
- **THEN** `ateam/handoff` is `success`
- **AND** when the checkpoint differs from the head or the tree was dirty it is `pending`, and with no handoff nothing is posted

#### Scenario: Review record
- **WHEN** the newest review record names the PR head
- **THEN** `ateam/review-record` is `success` for `APPROVE`, `failure` for `REQUEST CHANGES`, and `pending` for any other verdict
- **AND** when the record names another sha it is `pending`, and with no record nothing is posted

#### Scenario: Unchanged status
- **WHEN** a context's state and description equal the latest status with that context on the head sha
- **THEN** that context is not posted again

#### Scenario: Write failure
- **WHEN** a status write fails
- **THEN** handoff and sync print one warning and exit as they would without statuses
- **AND** `ateam doctor` warns from the cache that the last commit status write failed, prints no token, and `--check` still passes on that warning alone

#### Scenario: Disabled
- **WHEN** `ATEAM_COMMIT_STATUSES=0` is set, or a handoff runs with `--no-post` or `post: false`
- **THEN** no commit status is read or written
