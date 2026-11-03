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

### Requirement: Machine-readable writer lease
`ateam state writer --json` SHALL print the stored writer lease plus `live`, true for a lease with an agent whose PID is alive when set and whose `last_seen` is within the writer lease TTL when set, regardless of the caller's identity, and `exited`, true for a lease whose PID has exited. With no lease it SHALL print `null` and exit 0. An unparsable lease SHALL exit non-zero.

#### Scenario: Foreign live lease
- **WHEN** another session holds a recent lease with a live PID
- **THEN** the output has `live` true and `exited` false

#### Scenario: Exited writer
- **WHEN** the lease's PID has exited
- **THEN** the output has `live` false and `exited` true

#### Scenario: No lease
- **WHEN** no writer lease exists
- **THEN** the command prints `null` and exits 0
