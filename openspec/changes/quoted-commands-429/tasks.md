# Tasks: quoted-commands-429

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-27) sha256:1b9e66830cc2`), written by claude on the owner's instruction in the Enjoy conversation of 2026-11-27.

## 2. Implementation

- [ ] 2.1 FlagSet inventory (premise: 23 `flag.NewFlagSet` calls in `cmd/ateam/*.go`, all defining flags in the same function): `go/ast` walk recording path → flags and bool kind. Accept: table test asserting `review request` has bool `dry-run` and value `return-to`, `doctor` has bool `check`, `init` lacks `check`.
- [ ] 2.2 Quoted-command check over `AGENTS.md`, `.agents/protocol.md`, `.agents/contracts/*.md`, `.agents/skills/*/SKILL.md`. Accept: fixture table (flag after positional, unknown flag, trailing value-flag fragment accepted, `--x=v` accepted) plus the shipped-file pass; mutation check on the two rejection rules.
- [ ] 2.3 Preflight skill and template: `ateam init --check` → `ateam doctor --check`. Accept: bootstrap parity tests.
- [ ] 2.4 Spec scenario; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
