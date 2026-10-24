## 1. Implementation

- [ ] 1.1 `--force` writes `<name>.ateam-backup-<YYYYMMDD-HHMMSS>` for `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` and `.agents/rules-index.md` when their content differs from the template, reports the paths under `backed_up`, and fails without replacing a file whose backup cannot be written (#152).
- [ ] 1.2 Tests: force over a modified `AGENTS.md` leaves a backup with the original bytes and reports it; force over an unmodified template leaves none; a failed backup write leaves the original untouched. `task mutate:check` with the content comparison dropped reports `detection` (#152).
- [ ] 1.3 `docs/adoption.md` names the backup where it describes `--force` (#152).
