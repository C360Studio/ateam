## ADDED Requirements

### Requirement: Doctor checks the verify target
`ateam doctor` SHALL add a violation when the repository root has no `Taskfile.yml`, or has one whose task list, as `ateam task` discovers it, names no `verify` task, because the shipped skills and protocol run `task verify` by name.

#### Scenario: No Taskfile
- **WHEN** `ateam doctor --check` runs in a repository with no `Taskfile.yml` at its root
- **THEN** it reports the violation `Taskfile.yml: missing; the shipped skills and protocol run task verify` and exits non-zero

#### Scenario: Taskfile without a verify task
- **WHEN** the root `Taskfile.yml` defines `check` and no `verify`
- **THEN** it reports the violation `Taskfile.yml: no verify target; the shipped skills and protocol run task verify` and exits non-zero

#### Scenario: Verify task present
- **WHEN** the root `Taskfile.yml` defines `verify`, or includes a file that defines it in a way the Task executable lists as `verify`
- **THEN** doctor adds no verify-target violation
