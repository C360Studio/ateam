## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#168). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `InitRepo` removes a symbolic link at a forced write path before opening it, so `--force` never writes through the link (#168).
- [ ] 2.2 A test links `AGENTS.md` to a file outside the temp repo and checks the target is unchanged, the path is a regular file holding the template, and the backup holds the target's bytes. A mutation check shows it can fail (#168).
