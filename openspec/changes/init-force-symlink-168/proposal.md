# Change Proposal: `init-force-symlink-168`

Approved-by: Coby Leuschke (2026-10-27)

## Why

`ateam init --force` opens each generated file with `os.OpenFile(..., O_TRUNC)` in `InitRepo` (`internal/bootstrap/bootstrap.go`), and that call follows symbolic links. If an adopter's `AGENTS.md` links to a shared rules file elsewhere on the machine, `--force` truncates and rewrites the link's target, which may be outside the repository (#168). Since #167 the backup keeps the target's bytes, but the shared file is still changed.

## What Changes

- Before `--force` writes a generated file, init checks the path with `os.Lstat`. If it is a symbolic link, init removes the link and writes the template as a regular file at that path. The target is never opened for writing.
- The check covers every file `--force` replaces, not only the four harness documents, because all of them are opened the same way.
- The backup is unchanged: a symlinked harness document whose content differs from the template is backed up first, and the backup holds the target's bytes.
- The "Agent initialization and configuration parity" requirement in the harness spec gains one scenario.

## Track

`quick`. The change modifies one existing requirement and adds none. It touches only `internal/bootstrap` and adds no exported surface, boot path, or gate file. The backup order and sync stay as they are.

## Non-goals

- Refusing `--force` on a symlink. Replacing the link is what `--force` means, and the backup keeps the content.
- Symlinked parent directories (an `.agents/` that links elsewhere). `MkdirAll` and `OpenFile` follow them today; that is a separate issue if it matters.
- Closing the gap between `Lstat` and `OpenFile`. A local CLI run by the repo owner gains little from `O_NOFOLLOW`, which is not portable.
- Non-forced init. It already skips any existing path, including a link, through `Lstat` and `O_EXCL`.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green.
- A test in `internal/bootstrap/backup_test.go` links `AGENTS.md` to a file outside the temp repo. After `--force`, the target's bytes are unchanged, `AGENTS.md` is a regular file holding the template, and the backup holds the target's bytes. `task mutate:check` with the link removal dropped reports `detection`.
