## MODIFIED Requirements

### Requirement: Merge check runs on pull-request state events

`Merge check` SHALL run in its own workflow (`.github/workflows/merge-check.yml` and its Gitea twin), separate from `CI`, on the pull-request events that change its inputs, and SHALL be a required context of its own beside `Required` and `Gate files`. `Required` SHALL need only `Verify`. While the pull request description contains no `reviewed-by:`, the job SHALL be skipped under a name that is not a required context, as on a draft.

#### Scenario: The reviewer's description edit re-runs the merge check

- **WHEN** a ready code pull request's head carries a review record comment and the reviewer then edits the description to add `reviewed-by:`
- **THEN** the `edited` event runs `Merge check` against the head with no manual rerun, and it passes

#### Scenario: An open flake issue does not fail Required

- **WHEN** an issue labelled `class:flake` is open and a pull request that does not close it gets a green `Verify`
- **THEN** `Required` passes and `Merge check` fails, naming the open issue

#### Scenario: Pinned merge-check workflow

- **WHEN** `.github/workflows/merge-check.yml` is evaluated against the merge-gate contract
- **THEN** it triggers only on `pull_request` (`opened`, `synchronize`, `reopened`, `ready_for_review`, `edited`), runs one queued run per pull request (`concurrency.group` per pull request number, `cancel-in-progress: false`, so no cancelled `Merge check` sits beside a pass at one head), has the one job `merge-check` on `ubuntu-24.04` with only `contents: read`, `issues: read`, and `pull-requests: read`, and runs `scripts/merge-check.sh "$PR_NUMBER"` with `ATEAM_REQUIRED_CHECKS: Required,Merge check`, under `if: ${{ !github.event.pull_request.draft && contains(github.event.pull_request.body, 'reviewed-by:') }}` and named `Merge check (draft, skipped)` on a draft, `Merge check (awaiting review record, skipped)` without the line, and `Merge check` otherwise

#### Scenario: A draft pull request carries no Merge check

- **WHEN** a draft pull request is opened, pushed, or edited
- **THEN** the merge-check job is skipped under a name that is not a required context, as for `Verify` on a draft

#### Scenario: A ready pull request awaiting its review record carries no failed Merge check

- **WHEN** a ready code pull request is opened, marked ready, pushed, or edited and its description contains no `reviewed-by:`
- **THEN** the merge-check job is skipped under `Merge check (awaiting review record, skipped)`, no failed `Merge check` is posted, and the absent required context keeps the pull request from merging

#### Scenario: A rejection after an approval at the same head is delivered by a rerun

- **WHEN** a passing `Merge check` follows an approving record and description line, and a later record at the same head carries `Verdict: REQUEST CHANGES`
- **THEN** the reviewer reruns the whole latest merge-check run for that head (not `--failed`) and observes it failing on the rejecting record, or reports that it lacked permission to rerun
