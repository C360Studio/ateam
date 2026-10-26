# Change Proposal: `hook-scope-128`

## Why

An adopter that pins `.claude/settings.json` byte for byte (AIP's review manifest hashes it, #128) cannot let ateam write its hooks there. `ateam hook` itself needs no project settings file: it reads the event on stdin and takes `--event` and `--agent`. Claude Code also merges hooks from `.claude/settings.local.json` and `~/.claude/settings.json`, so the hooks can live outside the pinned file today. `ateam doctor` cannot see them: `checkHookConfigs` (`internal/bootstrap/bootstrap.go:680`) reads only `.claude/settings.json`, `.codex/hooks.json` and `.gemini/settings.json`, and reports `missing hook command for event …`, so `doctor --check` is red in such a checkout.

## What Changes

- An explicit knob, `ATEAM_HOOK_SCOPE=user` (environment, following the existing `ATEAM_*` settings), lets doctor accept the Claude hooks from `.claude/settings.local.json` or `~/.claude/settings.json` when the project `.claude/settings.json` does not carry them.
- Doctor reports which file satisfied each client's hook check, so the parity output stays honest.
- For Codex and Gemini, the knob reads their user-scope hook file only if the client supports one; otherwise doctor says the client has no user-scope hooks instead of passing.
- Unset, doctor behaves exactly as today.
- `docs/adoption.md` and the templated rules rows (`internal/bootstrap/templates.go:210`, `:249`) describe the knob.

## Non-goals

- Shipping ateam as a Claude Code plugin whose `hooks/hooks.json` carries the hooks (noted on #128 as the longer-term shape).
- Changing what `ateam init` writes.
- AIP asks 2 (#129, landed) and 3 (#130).
