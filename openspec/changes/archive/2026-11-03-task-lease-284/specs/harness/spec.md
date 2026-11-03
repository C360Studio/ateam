## ADDED Requirements

### Requirement: Optional external task lease display
When `.ateam/task-lease.json` or the path in `ATEAM_TASK_LEASE_FILE` exists, pickup and session-start SHALL print one line naming its issue, holder, expiry, and source. ateam SHALL NOT write, refresh, or clear the file, and the file SHALL NOT affect the writer lease, the handoff slot, or prune. A missing file SHALL change no output. A malformed file SHALL produce one warning and SHALL NOT fail pickup or the hook.

#### Scenario: Fresh and expired lease
- **WHEN** the file is valid
- **THEN** pickup prints `Task lease: issue #<n> holder=<holder> expires=<rfc3339> (source <source>)`
- **AND** the line ends with `expired` only when `expires_at` is past

#### Scenario: Path override
- **WHEN** `ATEAM_TASK_LEASE_FILE` holds an absolute or root-relative path
- **THEN** pickup reads that path instead of `.ateam/task-lease.json`

#### Scenario: No lease file
- **WHEN** the file does not exist
- **THEN** pickup output is byte-for-byte what it would be without this feature

#### Scenario: Malformed lease file
- **WHEN** the file is not a regular file, is oversized, is not valid JSON, lacks a required field, carries a control character in a value, or has an unparsable `expires_at`
- **THEN** pickup prints one `WARNING: task lease file unreadable (<reason>)` line whose reason does not echo file content
- **AND** pickup exits as it would without the file

#### Scenario: Lease values cannot gate the writer lease
- **WHEN** a valid file's `holder` is the text of the different-writer warning
- **THEN** session-start still writes the writer lease

### Requirement: Task lease token in handoff Addresses
When the current handoff's `Addresses` carries `aip-lease:<id>`, with a case-insensitive prefix, and a valid task lease file is present, pickup SHALL print `handoff lease <id> is not the current task lease <id2>` when the IDs differ exactly. It SHALL print nothing on a match or when either side is absent, and SHALL make no network call.

#### Scenario: Token mismatch
- **WHEN** the handoff names `AIP-LEASE:abc` and the file's `id` is `def`
- **THEN** pickup prints one mismatch line naming both IDs

#### Scenario: Token match
- **WHEN** the handoff's token equals the file's `id`
- **THEN** pickup prints no mismatch line

### Requirement: Machine-readable writer state
`ateam state writer --json` SHALL print one object with `state` (`none`, `live`, `stale`, or `unknown`), a fixed-text `reason`, and any stored writer fields, and SHALL exit 0 for every state. The first scenario below that matches a lease, in the order written, SHALL decide `state`. A PID check SHALL count only when the lease's non-empty `host` and its `pid_ns` equal the caller's. The result is advisory and SHALL NOT claim or change ownership.

#### Scenario: No lease
- **WHEN** no writer lease exists
- **THEN** `state` is `none`

#### Scenario: Unreadable lease
- **WHEN** the lease file cannot be parsed
- **THEN** `state` is `unknown`

#### Scenario: Incomplete lease
- **WHEN** the lease has no agent or no `last_seen`
- **THEN** `state` is `unknown` at any age

#### Scenario: Writer elsewhere within the TTL
- **WHEN** a lease within the writer lease TTL names another host, another PID namespace, or no host
- **THEN** `state` is `unknown` whatever the PID's status here

#### Scenario: Writer elsewhere past the TTL
- **WHEN** a lease past the writer lease TTL names another host, another PID namespace, or no host
- **THEN** `state` is `stale`

#### Scenario: No PID recorded
- **WHEN** a lease from this host and PID namespace has no PID
- **THEN** `state` is `unknown` at any age

#### Scenario: Exited writer
- **WHEN** a lease from this host and PID namespace names an exited PID
- **THEN** `state` is `stale`

#### Scenario: Long turn
- **WHEN** a lease from this host and PID namespace is past the TTL and its PID is alive
- **THEN** `state` is `unknown`

#### Scenario: Live writer
- **WHEN** a lease from this host and PID namespace is within the TTL and its PID is alive, whichever session holds it, including the caller or none
- **THEN** `state` is `live`

### Requirement: Machine-readable version and capabilities
`ateam version --json` SHALL print the version, the commit, and a `capabilities` list naming `state-writer-json` and `task-lease-file`. `ateam version` without the flag SHALL print its existing text.

#### Scenario: Capability list
- **WHEN** a client runs `ateam version --json`
- **THEN** the output parses as JSON and `capabilities` contains `state-writer-json`

## MODIFIED Requirements

### Requirement: Optional herdr pane identity in the writer lease
When `HERDR_ENV=1`, the writer lease SHALL record `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID` from the environment only, and pickup output SHALL name the pane in the writer line and the different-writer warning. Outside herdr the lease SHALL record no herdr identity and SHALL ignore herdr variables.

#### Scenario: Pane variable without herdr
- **WHEN** `HERDR_PANE_ID` is set but `HERDR_ENV` is not `1`
- **THEN** the lease records no herdr identity

#### Scenario: Different writer inside herdr
- **WHEN** a recent lease from another session carries a pane ID
- **THEN** the pickup warning names that pane
