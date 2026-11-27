# Proposal: quoted-commands-429
Approved-by: Coby Leuschke (2026-11-27) sha256:1b9e66830cc2

## Track

`quick`: one test in `cmd/ateam`, one MODIFIED Requirement scenario in the harness capability, and a one-line skill fix. No new command, flag, setting, or adopter-facing surface.

## Decisions you are making by approving

- `task verify` fails when a shipped instruction file quotes an `ateam` command that ateam's own argument parsing would reject: an unknown flag, a flag written after the first positional argument (the #426 shape), or a value flag missing its value in mid-command.
- The check reads ateam's own Go source to learn each subcommand's flags and never runs a command, so it makes no network call and has no side effects.
- It reads the files ateam ships to adopters: `AGENTS.md`, `.agents/protocol.md`, `.agents/contracts/*.md` and `.agents/skills/*/SKILL.md`. The bootstrap parity tests already tie these to the `ateam init` templates. `docs/` and `task <target>` lines are out of scope.
- A quoted fragment that ends on a value flag (`ateam review request --reviewer-session`) is accepted as a reference to that flag. The check does not count required positional arguments, which the source does not declare.
- The preflight skill's `ateam init --check`, which the check catches today (`flag provided but not defined: -check`), becomes `ateam doctor --check`.

## Why

#426 shipped because nothing parses the command lines quoted in shipped skills: the parity tests prove the skill and its template agree, not that the command parses. Measured on 2026-11-27 at 2077bd5, a second live case exists: `.agents/skills/preflight/SKILL.md` quotes `ateam init --check`, which exits 1.

## What Changes

1. A `cmd/ateam` test parses the package's non-test Go files with `go/ast`. For each `flag.NewFlagSet("<path>", …)` it records the flags defined on that set in the same function, and whether each is a bool flag.
2. It extracts every inline-code span starting with `ateam ` from the files above, splits it into tokens, and matches the longest FlagSet path. Tokens that follow the path are then checked against that set's flags with Go `flag` semantics: `--x=v` or `--x v` for value flags, a bare `--x` for bools, and no flag after the first positional argument. A path with no FlagSet must carry no flags.
3. `.agents/skills/preflight/SKILL.md` and its template: `ateam init --check` becomes `ateam doctor --check`.
4. Harness spec: one scenario added to "Agent initialization and configuration parity".

## Scope & Non-Goals

- **In-bounds:** `cmd/ateam/quoted_commands_test.go` (new), `.agents/skills/preflight/SKILL.md`, `internal/bootstrap/templates.go`, `openspec/specs/harness/spec.md`.
- **Non-goals:**
  - running commands or `--dry-run` sweeps;
  - `task` target lines;
  - `docs/`, the host manifests (gate files), and Go-generated message text;
  - positional arity;
  - an `ateam doctor` check for adopters. Adopters' copies come from the same templates, so ateam's CI covers them.

## Definition of Done

- The test fails on the pre-#428 coordinate line `ateam review request <pr> --return-to <your id>`, on `ateam init --check`, and on an unknown flag (fixture strings in a table test), and passes on the shipped files.
- `task mutate:check` detects removal of the flag-after-positional rule and of the unknown-flag rule.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before ready.
