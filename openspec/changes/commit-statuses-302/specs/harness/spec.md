## ADDED Requirements

### Requirement: Informational commit statuses on the PR head
`ateam state sync`, per open pull request whose branch has a local worktree, and a posting handoff (CLI or MCP), for its branch's open pull request, SHALL set the `ateam/proposal`, `ateam/handoff`, and `ateam/review-record` commit statuses on the PR head sha with the write token, never from a hook and never as a required check. A status SHALL be posted only when its state or description differs from the latest with its context on that sha. A failed write SHALL NOT change an exit code.

#### Scenario: Proposal approved or awaiting approval
- **WHEN** the PR touches an OpenSpec change whose `proposal.md` at the head has `Approved-by:` filled and no open `OPEN-Q` hold
- **THEN** `ateam/proposal` is `success` naming the approver
- **AND** with `Approved-by:` empty or the hold open it is `pending` `awaiting owner approval (OPEN-Q)`

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
