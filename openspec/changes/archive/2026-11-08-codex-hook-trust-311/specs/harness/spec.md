## ADDED Requirements

### Requirement: Doctor reports Codex project hook trust
When `.codex/hooks.json` satisfies doctor's Codex hook check, `ateam doctor` SHALL read the Codex config (`$CODEX_HOME/config.toml`, else `~/.codex/config.toml`) and report `codex_hook_trust` in its JSON output as `trusted`, `untrusted`, or `not checked: <reason>`. It SHALL add a warning, never a violation, when any required Codex hook event lacks a `hooks.state` entry keyed by this checkout's absolute `.codex/hooks.json` path.

#### Scenario: Checkout's hooks trusted
- **WHEN** the Codex config has a `hooks.state` key `<checkout>/.codex/hooks.json:<event>:…` for each of `session_start`, `user_prompt_submit`, `pre_compact`, and `stop`
- **THEN** doctor reports `codex_hook_trust: trusted` and adds no Codex trust warning

#### Scenario: Checkout's hooks not trusted
- **WHEN** one or more of those events has no key for this checkout's path, including when only another checkout's path (the main checkout, from a linked worktree) is trusted
- **THEN** doctor reports `codex_hook_trust: untrusted` and adds one warning naming the missing events and this checkout's path, telling the user to run `codex` here and approve the project hooks, and `--check` still passes on that warning alone

#### Scenario: Trust not checkable
- **WHEN** the Codex config is missing or is not valid TOML, or the Codex hooks were satisfied by `~/.codex/hooks.json` or failed doctor's hook check
- **THEN** doctor reports `codex_hook_trust: not checked: <reason>` and adds no Codex trust warning
