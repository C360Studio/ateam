## MODIFIED Requirements

### Requirement: Board state and authenticated API access

The board SHALL expose state and support card creation, movement, handoff, and mode updates through its API, with the static application served at the root. Without an API token, protected reads and writes SHALL accept only loopback clients; when a token is configured, every client SHALL supply it. Board helpers SHALL group cards into all six canonical columns, filter by text, agent, forge, and column, and format five-field pickup prompts deterministically.

The board SHALL support persisted repository registration, a selected repository, explicit forge settings, canonical OpenSpec requirement details and Current Ops. Mutations SHALL refuse a stale repository selection; failed persistence SHALL not change registration in memory. Explicit forge removal SHALL survive refresh, while automatic detection SHALL follow origin changes. Sync SHALL address the configured target and SHALL not fall back to a different repository after failure. Current Ops SHALL distinguish the selected checkout's inspected dirty state from uninspected worktrees, and compatible checkpoint SHA abbreviations SHALL not imply drift. Dashboard loads SHALL discard superseded results and clear stale state on failure. Dialogs SHALL contain focus, support Escape, and expose submission feedback. Benchmarks SHALL run on demand.

#### Scenario: Repository selection history

- **WHEN** a user adds and selects another repository, then removes it
- **THEN** state and model routing follow that selection and fall back to the startup repository
- **AND** stale clients cannot write to the newly selected repository

#### Scenario: Configured token

- **WHEN** a loopback client requests protected state without the configured token
- **THEN** the request is unauthorized
