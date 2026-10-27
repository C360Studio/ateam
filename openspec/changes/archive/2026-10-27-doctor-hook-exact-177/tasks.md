## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#177). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 `hookClient` names the native event key per required event, and `hookFileViolations` matches tokens of commands parsed under those keys, keeping the violation messages and removing the `// shortcut:` (#177). Evidence: `hookClient` gains `root`, `nested`, and `events []hookEvent{event, native}`; `hookFileViolations` reads `hookCommands(parsed[root][native], nested)`, and a command counts only when `isAteamHook` (first token `ateam` or `*/ateam`, then `hook`) and `hasFlag` finds exact `--event <ev>`; the agent violation fires when an event's commands exist but none carry `--agent <client>`.
- [x] 2.2 Sensitivity cases for `turn-ended`, `--agent` outside a hook command, and a command under the wrong native event; a mutation check reverting to substring matching shows each can fail (#177). Evidence: four new `TestHookScopeSensitivity` cases (`turn-ended`, a permissions-only `--agent claude`, `Stop` renamed `SubagentStop` at user scope, Gemini `AfterAgent` renamed `AfterModel` with the knob unset). `task mutate:check` with `-expect-text "expected violation containing"` under `/tmp/mut177`: the substring revert gave `detection` on all four; a prefix-matching `hasFlag` gave `detection` on `turn-ended`; searching every native key gave `detection` on both wrong-event cases (other pairs survive, as each case targets one rule). SHA-256 of `internal/bootstrap/bootstrap.go` (`b7ad84abfefbab38…`) was the same before and after.

Generated checks: not used. Inputs are hand-written JSON files with a closed set of event keys; each rule has a case that fails without it.
- [x] 2.3 Add the harness spec scenario, and have the tests cite the requirement (#177). Evidence: scenario "Hook commands match exactly under their native event" under "Hooks registered outside the project settings"; `TestHookScopeSensitivity` already cites that requirement.
