## ADDED Requirements

### Requirement: Approve an OpenSpec change from the board

`ateam serve` SHALL accept `POST /api/openspec/approve` with a change and its branch. It SHALL apply the owner approval edit to the files read at the branch head commit, and write it in one forge commit that is refused when `proposal.md`, `design.md` or `tasks.md` changed after that read, never to a local worktree or the default branch. The approver is the login behind the forge write token.

#### Scenario: Approve a change holding on OPEN-Q

- **WHEN** the owner approves a change whose `tasks.md` holds `Hold: OPEN-Q proposal awaits owner approval`
- **THEN** the branch gains the bound `Approved-by:` header for the read `proposal.md` and `design.md` and the hold is removed in the same commit

#### Scenario: Stale or unheld change

- **WHEN** `proposal.md`, `design.md` or `tasks.md` changed on the branch after it was read, or the change has no approval hold, or the branch is the default branch
- **THEN** the endpoint returns 409 for the stale file and 400 otherwise, and writes nothing
