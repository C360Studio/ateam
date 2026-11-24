## ADDED Requirements

### Requirement: Design-track trigger paths

When a repository declares `design_paths` globs, `ateam state spec-check` SHALL compare the changed paths against a base and fail when a changed path matches a glob and the diff adds or modifies no `design.md` under `openspec/changes/`. With no resolvable base, it SHALL print that the check did not run.

#### Scenario: Trigger path without a design

- **WHEN** the diff against the base touches a path matching a `design_paths` glob and no `openspec/changes/**/design.md`
- **THEN** spec-check exits non-zero, naming the path and the glob

#### Scenario: Trigger path with a design

- **WHEN** the same diff also adds or modifies a `design.md` under `openspec/changes/`
- **THEN** the design-paths check passes

#### Scenario: No base

- **WHEN** no `--base` is given and no default-branch merge base resolves
- **THEN** spec-check prints `design paths: not checked` and the check does not fail
