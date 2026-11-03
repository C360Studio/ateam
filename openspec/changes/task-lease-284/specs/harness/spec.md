## ADDED Requirements

### Requirement: Optional external task lease display
When `.ateam/task-lease.json` or the path in `ATEAM_TASK_LEASE_FILE` (absolute or root-relative) exists, pickup and session-start SHALL print one line naming its issue, holder, expiry, and source, with `expired` appended when `expires_at` is past. ateam SHALL NOT write, refresh, or clear the file, and the file SHALL NOT affect the writer lease, the handoff slot, or prune. When the file is absent, output SHALL be byte-for-byte unchanged. A file that is not a regular file, is oversized, is not valid JSON, lacks a required field, carries a control character in a value, or has an unparsable `expires_at` SHALL produce one `WARNING: task lease file unreadable (<reason>)` line, with a reason that does not echo file content, and SHALL NOT fail pickup or the hook.

#### Scenario: Fresh and expired lease
- **WHEN** the file is valid
- **THEN** pickup prints `Task lease: issue #<n> holder=<holder> expires=<rfc3339> (source <source>)`
- **AND** the line ends with `expired` only when `expires_at` is past

#### Scenario: No lease file
- **WHEN** the file does not exist
- **THEN** pickup prints no task lease line and no warning

#### Scenario: Malformed lease file
- **WHEN** the file is malformed
- **THEN** pickup prints one warning line and exits as it would without the file

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
`ateam state writer --json` SHALL print one object with `state` (`none`, `live`, `stale`, or `unknown`), a fixed-text `reason`, and the stored writer fields when a lease was read, and SHALL exit 0 for every state. A PID check SHALL count only when the lease's non-empty `host` and its `pid_ns` equal the caller's. `live` SHALL require a live PID within the writer lease TTL. `stale` SHALL mean past the TTL without a PID check saying otherwise, or an exited PID. Every other lease, including an unreadable one, a live PID past the TTL, or another host or PID namespace within the TTL, SHALL be `unknown`. The result is advisory and SHALL NOT claim or change ownership.

#### Scenario: No lease
- **WHEN** no writer lease exists
- **THEN** `state` is `none` and the command exits 0

#### Scenario: Foreign live lease
- **WHEN** another session on this host and PID namespace holds a recent lease with a live PID
- **THEN** `state` is `live` and the output names its session and host

#### Scenario: Exited writer
- **WHEN** a lease from this host and PID namespace names an exited PID
- **THEN** `state` is `stale`

#### Scenario: Writer elsewhere
- **WHEN** a recent lease names another host, another PID namespace, or no host
- **THEN** `state` is `unknown` whatever the PID's status here

#### Scenario: Long turn
- **WHEN** a lease from this host and PID namespace is past the TTL and its PID is alive
- **THEN** `state` is `unknown`

#### Scenario: Unreadable lease
- **WHEN** the lease file cannot be parsed
- **THEN** `state` is `unknown` and the command exits 0

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
