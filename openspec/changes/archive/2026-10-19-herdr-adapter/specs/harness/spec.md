## ADDED Requirements

### Requirement: Optional herdr pane identity in the writer lease
When `HERDR_ENV=1`, the writer lease SHALL record `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID` from the environment only, and pickup output SHALL name the pane in the writer line and the different-writer warning. Outside herdr the lease SHALL be byte-for-byte unchanged and SHALL ignore herdr variables.

#### Scenario: Pane variable without herdr
- **WHEN** `HERDR_PANE_ID` is set but `HERDR_ENV` is not `1`
- **THEN** the lease records no herdr identity

#### Scenario: Different writer inside herdr
- **WHEN** a recent lease from another session carries a pane ID
- **THEN** the pickup warning names that pane

### Requirement: Optional herdr caveat notifications
`ateam state herdr on-event` SHALL do nothing unless `HERDR_ENV=1` and `HERDR_BIN_PATH` are set. On `worktree.created` and non-working `pane.agent_status_changed` events in an ateam repository, it SHALL toast each `HALT`, `RED`, or `BLOCKED` caveat at most once per repository. Identity SHALL be the change, marker, and full caveat text, shared across path spellings and linked worktrees of one git common dir. The record SHALL be saved under a file lock before toasting. A failed toast SHALL NOT prevent the remaining claimed toasts from being attempted. It SHALL never run on the synchronous hook path.

#### Scenario: Gate closed
- **WHEN** `HERDR_ENV` is unset
- **THEN** herdr is never invoked

#### Scenario: Repeat and concurrent events
- **WHEN** the same caveat is seen by repeated or concurrent events, or from a linked worktree
- **THEN** exactly one toast is shown

#### Scenario: Partial delivery failure
- **WHEN** the first of two claimed toasts fails
- **THEN** the second is still attempted and the error is reported

### Requirement: herdr plugin panes
`ateam state herdr open-pane` SHALL accept only the `pickup` and `spec-queue` panes and open them through `herdr plugin pane open`. It SHALL pass the invoking pane's cwd and SHALL NOT pass a workspace, which herdr rejects for popups. Outside herdr it SHALL fail without invoking herdr.

#### Scenario: Action from a pane
- **WHEN** the action runs with `HERDR_PANE_ID` set
- **THEN** the popup opens with that pane's cwd

## MODIFIED Requirements

### Requirement: OpenSpec queue and stray Hold detection

`ateam state spec-queue` SHALL scan `openspec/changes/*/tasks.md` and report open-task holds (`Hold:`) and caveats (`WONTDO`, `HALT`, `RED`, `BLOCKED`, `OPEN-Q`). `ateam state spec-check` SHALL fail with exit code 1 when any `Hold:` appears below a file's first `## ` heading outside every task block, or exit code 2 when a `tasks.md` is unreadable or invalid UTF-8. Truncated caveat display text SHALL remain valid UTF-8.

#### Scenario: Stray Hold below heading

- **WHEN** an in-flight change's `tasks.md` has `Hold: waiting on #12` in a paragraph outside any `- [ ]` or `- [x]` task block below `## 1. Tasks`
- **THEN** `ateam state spec-check` reports `<path>:<line>:` and exits non-zero

#### Scenario: Multibyte text at the display cut

- **WHEN** a caveat's text crosses the display limit inside a multibyte character
- **THEN** the displayed text ends before that character and is valid UTF-8
