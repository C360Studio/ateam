## ADDED Requirements

### Requirement: Merge check runs on pull-request state events

`Merge check` SHALL run in its own workflow (`.github/workflows/merge-check.yml` and its Gitea twin), separate from `CI`, on the pull-request events that change its inputs, and SHALL be a required context of its own beside `Required` and `Gate files`. `Required` SHALL need only `Verify`.

#### Scenario: The reviewer's description edit re-runs the merge check

- **WHEN** a ready code pull request's head carries a review record comment and the reviewer then edits the description to add `reviewed-by:`
- **THEN** the `edited` event runs `Merge check` against the head with no manual rerun, and it passes

#### Scenario: An open flake issue does not fail Required

- **WHEN** an issue labelled `class:flake` is open and a pull request that does not close it gets a green `Verify`
- **THEN** `Required` passes and `Merge check` fails, naming the open issue

#### Scenario: A draft pull request carries no Merge check

- **WHEN** a draft pull request is opened, pushed, or edited
- **THEN** the merge-check job is skipped under a name that is not a required context, as for `Verify` on a draft
