## MODIFIED Requirements

### Requirement: Informational commit statuses on the PR head
`ateam state sync`, per open pull request with a local worktree, a posting handoff, for its branch's open pull request, and `ateam review notify`, for the commit its record names, SHALL set ateam's informational commit statuses with the write token, never from a hook and never as a required check. A status SHALL be posted only when it differs from the latest with its context on that sha. A failed write SHALL NOT change an exit code.

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

#### Scenario: Only documents or main merges after the record
- **WHEN** on GitHub every commit after the record's sha is a non-merge commit changing documents only, or a merge of `main` touching none of the PR's files, as `task merge:check` allows
- **THEN** `ateam/review-record` on the head carries the record's verdict state and names the record's sha
- **AND** a code commit, a merge touching a PR file, a commit missing from the checkout, or any later commit on Gitea leaves it `pending` behind head

#### Scenario: Reviewer notify
- **WHEN** a reviewer runs `ateam review notify --record <file>` with commit statuses enabled
- **THEN** `ateam/review-record` is set on the record's sha as if that sha were the head, and no other context is written
- **AND** `--decline` writes no status, and a failed write prints one warning while the notify message is still delivered

#### Scenario: Unchanged status
- **WHEN** a context's state and description equal the latest status with that context on the head sha
- **THEN** that context is not posted again

#### Scenario: Write failure
- **WHEN** a status write fails
- **THEN** handoff and sync print one warning and exit as they would without statuses
- **AND** `ateam doctor` warns from the cache that the last commit status write failed, prints no token, and `--check` still passes on that warning alone

#### Scenario: Disabled
- **WHEN** `ATEAM_COMMIT_STATUSES=0` or `commit_statuses: false` is set, or a handoff runs with `--no-post` or `post: false`
- **THEN** no commit status is read or written
