## 1. Implementation

- [ ] 1.1 With `ATEAM_HOOK_SCOPE=user`, `checkHookConfigs` accepts the Claude hooks from `.claude/settings.local.json` or `~/.claude/settings.json` when `.claude/settings.json` lacks them; unset, behaviour is unchanged (#128).
- [ ] 1.2 Doctor reports the file that satisfied each client's hook check. Codex and Gemini read a user-scope hook file only where the client supports one (verify against each client's documentation and record the source); otherwise doctor reports that the client has no user-scope hooks (#128).
- [ ] 1.3 Tests with a temp repo and a temp `HOME`: with the knob set and hooks only at user scope, doctor's hook checks pass and name the user file; with the knob unset they fail as today; a malformed or incomplete user file fails. `task mutate:check`: ignoring the knob, and accepting the user file without checking its hook commands, must each be `detection`.
- [ ] 1.4 `docs/adoption.md` and the templated rules rows in `internal/bootstrap/templates.go` describe the knob.
- [ ] 1.5 Move the requirement into `openspec/specs/harness/spec.md` and cite it there, then archive the change.
