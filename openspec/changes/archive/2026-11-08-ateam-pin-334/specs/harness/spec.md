## ADDED Requirements

### Requirement: Doctor checks a declared ateam pin

Outside the ateam checkout, when `.agents/ateam-version` exists, `ateam doctor` SHALL report the pin. It SHALL add a violation when the running binary's release, or the release of the binary the Claude hook command resolves to, differs from the pin by exact string. It SHALL also add a violation when the file does not hold one `v<major>.<minor>.<patch>` release. A hook binary that does not resolve SHALL be reported without a violation. Without the file, doctor's output SHALL be unchanged.

#### Scenario: Dev build against a release pin

- **WHEN** `.agents/ateam-version` holds `v0.3.9` and the hook command resolves to a binary reporting `v0.3.9-8-gbd6941f`
- **THEN** doctor reports a violation naming the hook binary's path, its version, and the pin, and exits non-zero

#### Scenario: Both binaries at the pin

- **WHEN** the pin is `v0.3.9` and both the running binary and the hook binary report `v0.3.9`
- **THEN** doctor adds no pin violation

#### Scenario: No hook binary on a CI runner

- **WHEN** the pin is `v0.3.9`, the running binary reports `v0.3.9`, and no hook binary resolves
- **THEN** doctor reports `hook_binary` as absent and adds no pin violation

#### Scenario: No pin declared

- **WHEN** `.agents/ateam-version` does not exist
- **THEN** doctor's output carries no pin field and no pin finding
