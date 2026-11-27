# Tasks: quoted-commands-429

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-27) sha256:1b9e66830cc2`), written by claude on the owner's instruction in the Enjoy conversation of 2026-11-27.

## 2. Implementation

- [x] 2.1 FlagSet inventory (premise: 23 `flag.NewFlagSet` calls in `cmd/ateam/*.go`, all defining flags in the same function): `go/ast` walk recording path → flags and bool kind. Accept: table test asserting `review request` has bool `dry-run` and value `return-to`, `doctor` has bool `check`, `init` lacks `check`.
- [x] 2.2 Quoted-command check over `AGENTS.md`, `.agents/protocol.md`, `.agents/contracts/*.md`, `.agents/skills/*/SKILL.md`. Accept: fixture table (flag after positional, unknown flag, trailing value-flag fragment accepted, `--x=v` accepted) plus the shipped-file pass; mutation check on the two rejection rules.
- [x] 2.3 Preflight skill: `ateam init --check` → `ateam doctor --check`. The `ateam init` preflight template has no such line, so only `.agents/skills/preflight/SKILL.md` changes. Accept: bootstrap parity tests.
- [x] 2.4 Spec scenario; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.

## Evidence

- Mutation (cp + SHA-256 restore, `quoted_commands_test.go` is a test file so `task mutate:check` cannot target it): dropping the flag-after-positional rule fails `TestCheckQuotedCommand` on the #426 line and the `--dry-run 425 --return-to` line (detection); dropping the unknown-flag rule fails it on `ateam init --check` and `--bogus` (detection); restored SHA matched.
- Examples suffice over generated checks: the grammar is a flat token walk with four outcomes, each with a table row, and the shipped-file pass covers 28 distinct real commands.
- Round 2 (codex REQUEST CHANGES at 7dd9019): receivers now resolve by declaration (`ast.Object`), so the two `fs :=` in `runExperimentCLI`'s switch cases stay separate; a set parsed from `args[N:]` (bench run/compare/export) skips N-1 leading positionals, with the inventory of such sets asserted. Mutation (cp + SHA-256): matching receivers by name fails the `bench run` inventory assertion; ignoring the parse offset fails the three valid bench rows (detection); SHA restored.
