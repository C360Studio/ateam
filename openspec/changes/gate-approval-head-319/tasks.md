## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#319). Hold: OPEN-Q proposal awaits owner approval

## 2. Fix

- [ ] 2.1 `scripts/gate-files.sh` (`:77-88`) requires, with the label, a newest `gate-change-approved: <sha>` comment naming the live head, on GitHub and Gitea (#319, design D1 B, D2). Acceptance: `TestGateFilesCheck` and `TestGateFilesCheckGitea` cases in `proposal.md`; `task mutate:check` reports `detection` with the head comparison removed.
- [ ] 2.2 `.agents/protocol.md` § Gate files, the `AGENTS.md` and `.agents/rules-index.md` gate-files rows, and `README.md:260` name the comment and the label (#319). Acceptance: `ateam doctor --check` row counts unchanged; `task spec:check` passes.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#319).
