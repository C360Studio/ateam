## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#168). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 `InitRepo` removes a symbolic link at a forced write path before opening it, so `--force` never writes through the link (#168). Evidence: in `InitRepo` (`internal/bootstrap/bootstrap.go`), a forced write `Lstat`s the path and `os.Remove`s a symlink before `OpenFile` with `O_TRUNC`. An `Lstat` error other than not-exist fails init. Backups still run first and read through the link.
- [x] 2.2 A test links `AGENTS.md` to a file outside the temp repo and checks the target is unchanged, the path is a regular file holding the template, and the backup holds the target's bytes. A mutation check shows it can fail (#168). Evidence: `TestForceInitReplacesSymlinkInsteadOfWritingThrough` in `internal/bootstrap/backup_test.go`. `ateam mutate` with the `os.Remove` dropped reported `detection` ("symlink target was rewritten"); the SHA-256 of `bootstrap.go` was the same before and after.
