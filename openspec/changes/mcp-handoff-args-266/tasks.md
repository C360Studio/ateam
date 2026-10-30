## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#266).
      Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Decode errors fail `execStateTool` (`internal/mcpserver/server.go:273`) and `execTaskTool` (`:395`) before any action runs. Acceptance: numeric `goal`, numeric `post`, and numeric `target` each return `isError: true` with no state change; `task mutate:check` dropping the state-tool decode check reports `detection`.
- [ ] 2.2 A handoff with a blank goal is refused before `state.Refresh` (`:292`). Acceptance: the seeded-checkpoint test from the proposal (missing, empty, and whitespace-only goal; `post` omitted; zero forge posts; cache and lease byte-identical), and a valid same-session replacement; `task mutate:check` dropping the goal check reports `detection`.
- [ ] 2.3 Add the `goal` description in `Tools()` (`:65`). Acceptance: the schema budget test passes, and the token count is recorded in the PR.
- [ ] 2.4 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green.
- [ ] 2.5 Archive the change before marking the PR ready (#266).
