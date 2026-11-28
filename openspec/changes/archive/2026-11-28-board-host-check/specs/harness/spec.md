## MODIFIED Requirements

### Requirement: Board state and authenticated API access

The board SHALL expose state and support card creation, movement, handoff, and mode updates through its API, with the static application served at the root. Without an API token, protected reads and writes SHALL accept only loopback clients that address a loopback host; when a token is configured, every client SHALL supply it. Board helpers SHALL group cards into all six canonical columns, filter by text, agent, forge, and column, and format five-field pickup prompts deterministically.

#### Scenario: Repository and dashboard behavior

- **WHEN** a user manages repositories and operates the dashboard
- **THEN** The board SHALL support persisted repository registration, a selected repository, explicit forge settings, canonical OpenSpec requirement details and Current Ops. Mutations SHALL refuse a stale repository selection; failed persistence SHALL not change registration in memory. Explicit forge removal SHALL survive refresh, while automatic detection SHALL follow origin changes. Sync SHALL address the configured target and SHALL not fall back to a different repository after failure. Current Ops SHALL distinguish the selected checkout's inspected dirty state from uninspected worktrees, and compatible checkpoint SHA abbreviations SHALL not imply drift. Dashboard loads SHALL discard superseded results and clear stale state on failure. Dialogs SHALL contain focus, support Escape, and expose submission feedback. Benchmarks SHALL run on demand.

#### Scenario: Repository selection history

- **WHEN** a user adds and selects another repository, then removes it
- **THEN** state and model routing follow that selection and fall back to the startup repository
- **AND** stale clients cannot write to the newly selected repository

#### Scenario: Tokenless request addressed to another host

- **WHEN** a loopback client sends a protected read or write with no token configured, and its `Host`, or its `Origin` when present, is not `localhost`, `127.0.0.1` or `[::1]`, with or without a port
- **THEN** the request is refused with 403 and writes nothing
- **AND** with a loopback `Host` and no foreign `Origin` it is served, and with a token configured the host is not checked

#### Scenario: Configured token

- **WHEN** a loopback client requests protected state without the configured token
- **THEN** the request is unauthorized

#### Scenario: Merged worktree badge in Current Ops

- **WHEN** Current Ops renders worktrees alongside synced pull-request cards and git branch ancestry
- **THEN** a worktree whose branch PR is merged (and has no newer unmerged commits ahead of `main` or the merged PR head) or whose branch is an ancestor of `main` with no open PR is marked `merged` (`Merged / potentially stale`) with its merge evidence (`Merged via PR #N` or `Branch ancestor of main`) while preserving local dirty status
- **AND** worktrees with an open PR, a closed-unmerged PR, new commits after a merged PR, or unavailable forge evidence are not marked `merged`

#### Scenario: In-flight change read view across worktrees

- **WHEN** a client reads `/api/openspec` while another worktree's branch holds an OpenSpec change on `Hold: OPEN-Q`
- **THEN** the response lists that change once with its branch, its worktree, its `proposal.md` text and its approval state read from that worktree's files
- **AND** the OpenSpec tab sorts changes holding on `OPEN-Q` first and links each to its branch's pull request when the synced cards hold one

### Requirement: Approve an OpenSpec change from the board

`ateam serve` SHALL accept `POST /api/openspec/approve` with a change and its branch, which SHALL be a valid git branch name. It SHALL apply the owner approval edit to the files read at the branch head commit, and write it in one forge commit that is refused when `proposal.md`, `design.md` or `tasks.md` changed after that read, never to a local worktree or the default branch. The approver is the login behind the forge write token.

#### Scenario: Approve a change holding on OPEN-Q

- **WHEN** the owner approves a change whose `tasks.md` holds `Hold: OPEN-Q proposal awaits owner approval`
- **THEN** the branch gains the bound `Approved-by:` header for the read `proposal.md` and `design.md` and the hold is removed in the same commit

#### Scenario: Stale or unheld change

- **WHEN** `proposal.md`, `design.md` or `tasks.md` changed on the branch after it was read, or the change has no approval hold, or the branch is the default branch
- **THEN** the endpoint returns 409 for the stale file and 400 otherwise, and writes nothing

#### Scenario: Branch that is not a git branch name

- **WHEN** the branch is `@` or `HEAD`, starts with `-`, holds `..`, `@{`, a space, a control character or any of `~ ^ : ? * [ \ #`, or has a slash-separated part that is empty, starts with `.`, or ends with `.` or `.lock`
- **THEN** the endpoint returns 400 before any forge call and writes nothing
