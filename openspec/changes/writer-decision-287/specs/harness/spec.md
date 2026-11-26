## ADDED Requirements

### Requirement: Session start takes the writer lease from the lease itself

Session-start SHALL decide whether another writer is active from the cached writer lease, never from rendered text, and SHALL print the different-writer warning from the same decision. A lease from another host, another PID namespace, or no host, within the writer lease TTL, SHALL count as another writer without a PID check. When another writer is active, session-start SHALL NOT write the writer lease, and before-prompt and turn-end SHALL NOT refresh it.

#### Scenario: Cached text carrying the warning

- **WHEN** a cached decision or handoff field holds the different-writer warning text on its own line and no other writer is active
- **THEN** session-start writes the writer lease

#### Scenario: Recent lease from elsewhere

- **WHEN** another session's lease within the TTL names another host or PID namespace, whatever its PID's status here
- **THEN** session-start prints the warning, and that lease stays in place through session-start, before-prompt and turn-end

#### Scenario: Old lease from elsewhere

- **WHEN** another session's lease past the TTL names another host or PID namespace
- **THEN** session-start prints no warning and writes the writer lease
