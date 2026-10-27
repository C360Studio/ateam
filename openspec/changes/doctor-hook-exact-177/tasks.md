## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#177). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `hookClient` names the native event key per required event, and `hookFileViolations` matches tokens of commands parsed under those keys, keeping the violation messages and removing the `// shortcut:` (#177).
- [ ] 2.2 Sensitivity cases for `turn-ended`, `--agent` outside a hook command, and a command under the wrong native event; a mutation check reverting to substring matching shows each can fail (#177).
- [ ] 2.3 Add the harness spec scenario, and have the tests cite the requirement (#177).
