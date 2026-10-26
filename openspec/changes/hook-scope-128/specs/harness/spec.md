## ADDED Requirements

### Requirement: Hooks registered outside the project settings

`ateam doctor` SHALL accept ateam's hooks registered outside the project's client settings files only when `ATEAM_HOOK_SCOPE=user` is set, and SHALL report which file satisfied each client's hook check.

#### Scenario: Claude hooks at user scope

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and `.claude/settings.json` has no ateam hooks, but `.claude/settings.local.json` or `~/.claude/settings.json` carries every required `ateam hook --event <event> --agent claude` command
- **THEN** doctor reports no Claude hook violation and names the file that carried the hooks

#### Scenario: Knob unset

- **WHEN** `ATEAM_HOOK_SCOPE` is unset and `.claude/settings.json` has no ateam hooks
- **THEN** doctor reports `missing hook command for event <event>` for `.claude/settings.json`, as before, whatever the user-scope files contain

#### Scenario: Client without user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and a client's project hook file lacks the ateam hooks and the client has no user-scope hook file
- **THEN** doctor reports that the client has no user-scope hooks, rather than passing
