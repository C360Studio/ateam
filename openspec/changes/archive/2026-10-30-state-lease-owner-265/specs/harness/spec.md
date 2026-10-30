## ADDED Requirements

### Requirement: Handoff releases only the caller's writer lease

`ateam state handoff` and `ateam_state(action="handoff")` SHALL remove `.ateam/writer.json` only when the lease is live and positively identifies the caller by session ID, or, without one, by herdr pane or client/own PID. The ownership decision, the save, and the removal SHALL run under the state lock that every lease write holds. Any other lease SHALL stay byte-identical and SHALL remain the cached writer.

#### Scenario: Another session's live lease survives a handoff

- **WHEN** `writer.json` holds a live lease for session A (any agent) and session B saves a handoff through the CLI or MCP
- **THEN** the handoff is saved, `writer.json` is byte-identical, and the cached writer equals it

#### Scenario: The caller's own lease is released

- **WHEN** `writer.json` holds a live lease for the caller's session and the handoff succeeds
- **THEN** `writer.json` is removed and the cached writer is empty

#### Scenario: A refused handoff releases nothing

- **WHEN** the handoff is refused because another session's fresh handoff holds the slot
- **THEN** neither `.ateam/state.json` nor `writer.json` changes

#### Scenario: A lease written during the handoff survives

- **WHEN** a lease write starts after the handoff has decided ownership and before it would remove the lease
- **THEN** the write waits for the state lock, and the replacement lease exists after the handoff returns

#### Scenario: A failed release is reported

- **WHEN** the handoff is saved but `writer.json` cannot be removed
- **THEN** the CLI exits 1 and MCP returns an error, each saying that the handoff was saved and the writer lease was not released
