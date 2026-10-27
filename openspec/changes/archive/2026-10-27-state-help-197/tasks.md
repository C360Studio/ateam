## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#197). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 `runState` prints state usage and exits 0 for `-h`, `--help`, and `help` before any state access. An unknown subcommand points to `ateam state --help` (#197). Evidence: a `case "-h", "--help", "help"` in `runState` prints the `stateUsage` const and returns before any `state` call. The default case now ends `; run 'ateam state --help' for subcommands`.
- [x] 2.2 A test covers help exit status, the listed subcommands, no state written, and the unchanged unknown-subcommand failure. A mutation check shows it can fail (#197). Evidence: `TestStateHelpIsReadOnly` in `cmd/ateam/main_test.go` runs each spelling in an empty temp root. Each must exit 0, list all 11 subcommands, and leave no `.ateam`. `nope` must exit 1 with the help path. Five mutants under `/tmp/mut197` each gave `detection`: help case removed, `-h` dropped, hint removed, help that refreshes state, and a usage line removed. The SHA-256 of `cmd/ateam/main.go` (`22f3a0f6270c8964…`) was the same before and after.

Generated checks: not used. The input is one token matched against a closed set, and the table covers each help spelling plus one unknown token.
- [x] 2.3 Add the harness spec scenario, and have the test cite the requirement (#197). Evidence: scenario "State help is read-only" under "Durable shared state and five-field handoff". `TestStateHelpIsReadOnly` cites that requirement.
