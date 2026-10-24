# Change Proposal: `force-backup-152`

## Why

`ateam init --force` truncates `AGENTS.md` and `CLAUDE.md` and writes the ateam templates over them (`internal/bootstrap/bootstrap.go`, `preserve := !opts.Force || isClientConfig(rel)`). In an adopter's repo those are the files the adopter certainly wrote by hand, and nothing is kept (#152).

## What Changes

- Before `--force` replaces `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md`, the existing content is written to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>` beside it, if it differs from the template. If the backup cannot be written, the file is not replaced and init fails.
- The init JSON lists the backup paths under `backed_up`.
- `docs/adoption.md` names the backup where it describes `--force`.
- No flag disables the backup.

## Non-goals

- Merging the adopter's rules into the template. The adopter reads the backup and moves their own lines.
- Trimming the gitignore lines that init appends (noted on #152, not part of this change).
