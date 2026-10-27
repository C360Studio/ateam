## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#197). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `runState` prints state usage and exits 0 for `-h`, `--help`, and `help` before any state access. An unknown subcommand points to `ateam state --help` (#197).
- [ ] 2.2 A test covers help exit status, the listed subcommands, no state written, and the unchanged unknown-subcommand failure. A mutation check shows it can fail (#197).
- [ ] 2.3 Add the harness spec scenario, and have the test cite the requirement (#197).
