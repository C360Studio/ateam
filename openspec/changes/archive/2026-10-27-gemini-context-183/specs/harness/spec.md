## MODIFIED Requirements

### Requirement: Agent initialization and configuration parity

Initialization SHALL write usable canonical instructions, skill adapters, hook settings, and OpenSpec scaffolding. Parity checks SHALL reject missing AGENTS imports, instruction size overflow, mismatched rule row counts, copied adapter checklists, missing native hook configuration, placeholder workflow content, and documented hook event drift.

#### Scenario: Generated configuration

- **WHEN** a repository is initialized with default templates
- **THEN** its agent configuration passes parity checks

#### Scenario: Shipped review-pr skill

- **WHEN** a repository is initialized with default templates
- **THEN** `.agents/skills/review-pr/SKILL.md` is written with the same body this repository uses, with thin adapters at `.claude/skills/review-pr/SKILL.md` and `.gemini/commands/review-pr.toml`, and parity checks reject a canonical `review-pr` skill that loses its numbered steps or the record grammar markers `reviewed-by:` and `Reviewed at`

#### Scenario: Build provenance and duplicate server detection in doctor

- **WHEN** `ateam version`, `session-start`, or `ateam doctor` runs
- **THEN** build provenance (`vcs.revision` or `-ldflags` commit) is reported in `ateam version` and the `[ateam]` session-start line (or `unknown` when built without VCS metadata), and `ateam doctor` lists running `ateam serve` and `ateam mcp` processes (with their build commits) and warns when the binary has no build provenance (`binary has no build provenance; build with task build`), when the binary commit is behind checkout `HEAD` (`binary <build> is behind HEAD <head>; run task build`) or not an ancestor of checkout `HEAD`, when hook latency exceeds `10000 us` (`<10ms` budget), when a running `ateam mcp` process is stale, or when multiple `ateam serve` processes are running concurrently

#### Scenario: Stale or missing binary build provenance warning at session start

- **WHEN** `ateam hook --event session-start` runs in the ateam checkout (root `go.mod` declares `module github.com/c360studio/ateam`) with a resolved `HEAD` commit
- **THEN** it compares the embedded build commit (`state.BuildRevision()`) against `ReadFastGitHead` without spawning a git subprocess, emits `binary <build> is behind HEAD <head>; run task build` when the binary commit differs from `HEAD`, emits `binary has no build provenance; build with task build` when build provenance is `unknown` or `dev`, and emits neither line when the binary commit matches `HEAD`

#### Scenario: Build provenance checks only the ateam checkout

- **WHEN** the session-start hook or `ateam doctor` runs in a checkout whose root `go.mod` does not declare `module github.com/c360studio/ateam` (or that has no `go.mod`)
- **THEN** neither compares the binary's build commit with the checkout's `HEAD`: the hook prints neither a `binary <build> is behind HEAD` line nor a `binary has no build provenance` line, doctor prints no not-an-ancestor warning and reports `build_provenance: not applicable (binary is not built from this checkout)`, and the module path is read from `go.mod` with a line scan, without spawning a subprocess
- **AND** in the ateam checkout both checks behave as before

#### Scenario: Forced init backs up hand-written harness documents

- **WHEN** `ateam init --force` would replace `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md` and the existing content differs from the template
- **THEN** the existing bytes are first written to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>` beside the file, and the init result lists each backup path under `backed_up`
- **AND** every backup is written, synced and closed before any file is written, so if any backup cannot be written init fails without replacing any file, and a partly written backup is removed
- **AND** a file whose content already equals the template, and any generated file other than those four, gets no backup
- **AND** when init fails after making backups, it still reports their paths, and `backed_up` is an empty list rather than `null` when nothing was backed up

#### Scenario: Forced init replaces a symlinked file instead of writing through it

- **WHEN** `ateam init --force` would replace a generated file whose path is a symbolic link, for example an `AGENTS.md` that links to a shared rules file outside the repository
- **THEN** init removes the link and writes the template as a regular file at that path, so the link's target is never opened for writing and keeps its bytes
- **AND** a symlinked harness document whose target differs from the template is still backed up first, and the backup holds the target's bytes

#### Scenario: Gemini context file uses the nested key

- **WHEN** `ateam init` writes `.gemini/settings.json`, or doctor checks a Gemini settings file
- **THEN** init writes `"context": {"fileName": "AGENTS.md"}` and no top-level `contextFileName`, because Gemini CLI reads only the nested key
- **AND** doctor accepts only a `context.fileName` of `AGENTS.md`; a file that names `AGENTS.md` only in the top-level `contextFileName` fails with `missing context.fileName AGENTS.md (Gemini CLI ignores the top-level contextFileName)`

### Requirement: Hooks registered outside the project settings

`ateam doctor` SHALL accept ateam's hooks registered outside the project's client settings files only when `ATEAM_HOOK_SCOPE=user` is set, and SHALL report which file satisfied each client's hook check under `hook_sources` in its JSON output (an object keyed by client agent, `{}` when none passed).

#### Scenario: Claude hooks at user scope

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and `.claude/settings.json` has no ateam hooks, but `.claude/settings.local.json` or `~/.claude/settings.json` (read in that order) carries every required `ateam hook --event <event>` command and `--agent claude`
- **THEN** doctor reports no Claude hook violation and names the file that carried the hooks; Codex reads `~/.codex/hooks.json` and Gemini `~/.gemini/settings.json` the same way

#### Scenario: Partial, malformed or split local or user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set, the project file lacks the hooks, and no single local or user-scope file is valid JSON carrying every required event and the client's `--agent` (hooks split across files count as missing)
- **THEN** doctor reports the project file's violations and each local or user-scope file's, for example `~/.claude/settings.json: invalid JSON` or `~/.claude/settings.json: missing hook command for event <event>`

#### Scenario: Knob unset

- **WHEN** `ATEAM_HOOK_SCOPE` is unset and `.claude/settings.json` has no ateam hooks
- **THEN** doctor reports `missing hook command for event <event>` for `.claude/settings.json`, as before, whatever the local and user-scope files contain

#### Scenario: Gemini context.fileName

- **WHEN** `.gemini/settings.json` sets its context file (the nested `context.fileName`; the top-level `contextFileName` is not read) to anything other than `AGENTS.md`, or sets none while either the knob is unset or `~/.gemini/settings.json` is missing, invalid or lacks `context.fileName: "AGENTS.md"`
- **THEN** doctor reports `<file>: missing context.fileName AGENTS.md` naming the file whose value was checked (the project file when it sets a value, as Gemini gives project settings precedence; `~/.gemini/settings.json` when the knob is set, the project file sets none, and that file parses), whichever file carried the hooks

#### Scenario: Unknown knob value

- **WHEN** `ATEAM_HOOK_SCOPE` is set to anything other than `user` (compared case-insensitively, surrounding space ignored)
- **THEN** doctor reports `ATEAM_HOOK_SCOPE=<value>: unknown value; set it to user or unset it` and reads only the project files

#### Scenario: Client without user-scope hooks

- **WHEN** `ATEAM_HOOK_SCOPE=user` is set and a client's project hook file lacks the ateam hooks and doctor reads no user-scope hook file for that client (Antigravity's `.agents/hooks.json`)
- **THEN** doctor reports that the client has no user-scope hooks doctor reads (`.agents/hooks.json: ATEAM_HOOK_SCOPE=user, but agy has no user-scope hooks doctor reads`), rather than passing

#### Scenario: Hook commands match exactly under their native event

- **WHEN** a hook file's only command for a required event names a different event (`--event turn-ended` for `turn-end`), sits under a native event key other than the one that maps to that event (a `turn-end` command under `SessionStart`), or carries the client's `--agent` only outside a hook command (a permissions entry `Bash(ateam hook --agent claude:*)` while the hooks say `--agent codex`)
- **THEN** doctor reports `missing hook command for event <event>` or `missing --agent <client> in hook commands` for that file and does not name it under `hook_sources`, with or without `ATEAM_HOOK_SCOPE=user`
