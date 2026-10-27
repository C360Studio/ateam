# Change Proposal: `doctor-hook-exact-177`

Approved-by: Coby Leuschke (2026-10-27)

## Why

`ateam doctor` accepts a hook file when its raw text contains `ateam hook --event <ev>` for each required event and `--agent <client>` anywhere (#177, from the #176 review). Three wrong files pass `doctor --check` with rc=0: `--event turn-ended` satisfies `turn-end`, so no turn-end hook runs; a permissions entry `Bash(ateam hook --agent claude:*)` satisfies `--agent claude` while every hook says `--agent codex`; and a command under the wrong native event key (for example `turn-end` under `SessionStart`) counts. With `ATEAM_HOOK_SCOPE=user` the same check reads user-scope files people edit by hand. `hookFileViolations` in `internal/bootstrap/bootstrap.go` carries a `// shortcut:` naming this fix.

## What Changes

- `hookFileViolations` parses the hook commands in each client's schema instead of matching raw text: `hooks.<NativeEvent>[].hooks[].command` for Claude, Codex and Gemini, and `ateam.<NativeEvent>[].command` for Antigravity.
- Each `hookClient` names the native event key for every required event, taken from the templates `ateam init` writes (Claude and Codex: `SessionStart`, `UserPromptSubmit`, `PreCompact`, `Stop`; Gemini: `SessionStart`, `BeforeAgent`, `PreCompress`, `AfterAgent`; Antigravity: `SessionStart`, `PreInvocation`, `Stop`).
- A required event passes only when a command under its native event key splits (on white space) into `ateam hook` (the `ateam` token may be a path ending in `/ateam`) with the exact pairs `--event <ev>` and `--agent <client>`.
- The violation messages stay the same: `missing hook command for event <ev>` when no command for that event exists under its key, and `missing --agent <client> in hook commands` when ateam hook commands exist but none for the event carry the client's agent.
- The "Hooks registered outside the project settings" requirement gains one scenario for exact matching. The `// shortcut:` comment is removed.

## Track

`quick`. It modifies one existing requirement and adds none, touches one capability, and changes only unexported code in `internal/bootstrap` (no exported surface, boot path, durability, or gate file).

## Non-goals

- `--event=<ev>` or quoted forms, shell parsing, or commands wrapped in another program (`sh -c`, `env`); `ateam init` writes none of these.
- Reading Antigravity user-scope hooks, or a pre-compact hook for Antigravity (existing shortcuts).
- Checking hook `type`, `matcher`, or `timeout` fields.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green on this repo.
- `TestHookScopeSensitivity` (or a sibling citing the same requirement) gains the three cases from #177: `turn-ended`, `--agent` only outside a hook command, and a command under the wrong native event. Each must report a violation and leave the client out of `hook_sources`.
- `task mutate:check` reverting to substring matching shows `detection` for each case.
