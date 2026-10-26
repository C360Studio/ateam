## ADDED Requirements

### Requirement: Hooks registered outside the project settings

`ateam doctor` SHALL accept ateam's hooks registered outside the project's client settings files only when `ATEAM_HOOK_SCOPE=user` is set, and SHALL report which file satisfied each client's hook check under `hook_sources` in its JSON output (an object keyed by client agent, `{}` when none passed).

#### Scenario: Claude hooks at user scope

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and `.claude/settings.json` has no ateam hooks, but `.claude/settings.local.json` or `~/.claude/settings.json` (read in that order) carries every required `ateam hook --event <event>` command and `--agent claude`
- **THEN** doctor reports no Claude hook violation and names the file that carried the hooks; Codex reads `~/.codex/hooks.json` and Gemini `~/.gemini/settings.json` the same way, and Gemini's `contextFileName` may come from the project file or from the user file that carried the hooks

#### Scenario: Partial, malformed or split user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set, the project file lacks the hooks, and no single user-scope file is valid JSON carrying every required event and the client's `--agent` (hooks split across files count as missing)
- **THEN** doctor reports the project file's violations and each user-scope file's, for example `~/.claude/settings.json: invalid JSON` or `~/.claude/settings.json: missing hook command for event <event>`

#### Scenario: Knob unset

- **WHEN** `ATEAM_HOOK_SCOPE` is unset and `.claude/settings.json` has no ateam hooks
- **THEN** doctor reports `missing hook command for event <event>` for `.claude/settings.json`, as before, whatever the user-scope files contain

#### Scenario: Unknown knob value

- **WHEN** `ATEAM_HOOK_SCOPE` is set to anything other than `user` (compared case-insensitively, surrounding space ignored)
- **THEN** doctor reports `ATEAM_HOOK_SCOPE=<value>: unknown value; set it to user or unset it` and reads only the project files

#### Scenario: Client without user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and a client's project hook file lacks the ateam hooks and doctor reads no user-scope hook file for that client (Antigravity's `.agents/hooks.json`)
- **THEN** doctor reports that the client has no user-scope hooks doctor reads (`.agents/hooks.json: ATEAM_HOOK_SCOPE=user, but agy has no user-scope hooks doctor reads`), rather than passing
