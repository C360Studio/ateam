## ADDED Requirements

### Requirement: Read another agent's handoff without changing ownership

Pickup SHALL show an attributed, bounded goal preview for every other handoff slot it lists, and `ateam state read-handoff <agent>@<branch>` and `ateam_state(action="read_handoff", slot=...)` SHALL print one named slot's five fields without writing `.ateam/state.json`, touching the writer lease, changing the caller's agent or session, or posting to a forge.

#### Scenario: Pickup previews another agent's goal

- **WHEN** pickup runs as `codex` on `main` and the cache holds a live `claude@main` handoff
- **THEN** pickup prints `Last Handoff: none for codex@main` and `other handoffs: claude@main (<age>[, PR #<n>]): <goal>`, the goal clipped to 104 bytes on a word boundary with `…`

#### Scenario: Reading a named slot is read-only

- **WHEN** a Codex session runs `ateam state read-handoff claude@main` or `ateam_state(action="read_handoff", slot="claude@main")`
- **THEN** it prints the slot's five-field prompt, looking in the current checkout's cache and then linked worktrees' caches
- **AND** the cache bytes, the writer lease, and the caller's resolved agent and session are unchanged, and no forge call is made

#### Scenario: Stale and missing slots

- **WHEN** the named slot is stale under the handoff staleness rules
- **THEN** its fields are printed under `STALE — <reasons>` and the command exits 0
- **AND WHEN** no cache holds the named slot, the command exits 1 (MCP tool error) and lists the slot keys that exist
