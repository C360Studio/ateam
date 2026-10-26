## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#182).

## 2. Implementation

- [ ] 2.1 Export the `proposal.md` header parser from `internal/state` (#182).
- [ ] 2.2 `gateSource.fileAt` on GitHub and Gitea, with the fake-forge fixture answering it (#182).
- [ ] 2.3 Collect touched change directories on a code PR and print the note; read errors are notes, never unavailable (#182).
- [ ] 2.4 Tests: unapproved, approved, archived path, missing proposal, read error; mutation checks with `task mutate:check` (#182).
- [ ] 2.5 Harness spec scenario (#182).
