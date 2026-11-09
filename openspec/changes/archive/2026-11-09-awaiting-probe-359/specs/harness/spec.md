## MODIFIED Requirements

### Requirement: Merge check runs on pull-request state events

`Merge check` SHALL run in its own workflow (`.github/workflows/merge-check.yml` and its Gitea twin), separate from `CI`, on the pull-request events that change its inputs, and SHALL be a required context of its own beside `Required` and `Gate files`. `Required` SHALL need only `Verify`. While a code pull request awaits its review (no `reviewed-by:` line, no passing `Merge check` at its head), the job SHALL be skipped under a name that is not a required context, decided by a probe job built from the base.

#### Scenario: The reviewer's description edit re-runs the merge check

- **WHEN** a ready code pull request's head carries a review record comment and the reviewer then edits the description to add `reviewed-by:`
- **THEN** the `edited` event runs `Merge check` against the head with no manual rerun, and it passes

#### Scenario: An open flake issue does not fail Required

- **WHEN** an issue labelled `class:flake` is open and a pull request that does not close it gets a green `Verify`
- **THEN** `Required` passes and `Merge check` fails, naming the open issue

#### Scenario: Pinned merge-check workflow

- **WHEN** `.github/workflows/merge-check.yml` is evaluated against the merge-gate contract
- **THEN** it triggers only on `pull_request` (`opened`, `synchronize`, `reopened`, `ready_for_review`, `edited`), runs one queued run per pull request (`concurrency.group` per pull request number, `cancel-in-progress: false`, so no cancelled `Merge check` sits beside a pass at one head), has the two jobs `awaiting` and `merge-check` on `ubuntu-24.04` with only `contents: read`, `issues: read`, and `pull-requests: read`, each built from the base commit; `awaiting` is named `Merge check probe`, skips drafts, and publishes `ateam review gate --awaiting "$PR_NUMBER"`'s `awaiting=` answer as its output; `merge-check` needs it and runs `scripts/merge-check.sh "$PR_NUMBER"` with `ATEAM_REQUIRED_CHECKS: Required,Merge check`, under `if: ${{ always() && !github.event.pull_request.draft && needs.awaiting.outputs.awaiting != 'true' }}` and a `name:` expression that yields `Merge check (draft, skipped)` on a draft, `Merge check (awaiting review record, skipped)` while awaiting, and `Merge check` otherwise, though GitHub shows a skipped job's raw expression instead

#### Scenario: A draft pull request carries no Merge check

- **WHEN** a draft pull request is opened, pushed, or edited
- **THEN** the merge-check job is skipped under a name that is not a required context, as for `Verify` on a draft

#### Scenario: A ready pull request awaiting its review record carries no failed Merge check

- **WHEN** a ready code pull request is opened, marked ready, pushed, or edited, its description has no `reviewed-by:` line, and no successful `Merge check` stands at its head, whether or not a review record comment exists at that head or an earlier one
- **THEN** the merge-check job is skipped under a name other than `Merge check` (on GitHub, the raw `name:` expression, because GitHub does not evaluate a skipped job's `name:`), no failed `Merge check` is posted, and the absent required context keeps the pull request from merging

#### Scenario: The probe reads the forge and fails closed

- **WHEN** the `awaiting` job runs `ateam review gate --awaiting` for a ready pull request
- **THEN** it answers `awaiting=true` only for a code pull request with no `reviewed-by:` line and no successful `Merge check` at its head (GitHub check run, Gitea commit status, matched by exact name), reading the forge when it runs so a rerun sees the current state; a documents-only pull request, one with a line, or one with a standing pass at its head is judged in full, and a failed probe leaves no answer, so `Merge check` runs in full

#### Scenario: Removing the line after an approval fails instead of keeping the pass

- **WHEN** a passing `Merge check` stands at a head and the `reviewed-by:` line is then removed from the description without a push
- **THEN** the `edited` event runs `Merge check` in full, because a successful `Merge check` stands at the head, and it fails on the missing line

#### Scenario: A documents-only pull request is judged without a review

- **WHEN** a ready pull request changes only documents and has no `reviewed-by:` line and no review record
- **THEN** `Merge check` runs in full and passes its review check, as `ateam review gate` passes a documents-only pull request

#### Scenario: A rejection after an approval at the same head is delivered by a rerun

- **WHEN** a passing `Merge check` follows an approving record and description line, and a later record at the same head carries `Verdict: REQUEST CHANGES`
- **THEN** the reviewer reruns the whole latest merge-check run for that head (not `--failed`) and observes it failing on the rejecting record, or reports that it lacked permission to rerun
