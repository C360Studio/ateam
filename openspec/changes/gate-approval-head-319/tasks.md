## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#319). Evidence: the owner selected "Approve B (comment with sha + label)" in Enjoy on 2026-11-08; Claude wrote the header at their request.

## 2. Fix

- [x] 2.1 `scripts/gate-files.sh` (`:77-88`) requires, with the label, a newest `gate-change-approved: <sha>` comment naming the live head, on GitHub and Gitea (#319, design D1 B, D2). Acceptance: `TestGateFilesCheck` and `TestGateFilesCheckGitea` cases in `proposal.md`; `task mutate:check` reports `detection` with the head comparison removed. Evidence: `ateam mutate` takes only Go files, so the `cp` + SHA-256 method ran: with `"$approved"*)` replaced by `*)`, `TestGateFilesCheck` and `TestGateFilesCheckGitea` failed on `covers <old>, not head <head>`; the script's SHA-256 b0d3693b… was unchanged after restoring it.
- [x] 2.2 `.agents/protocol.md` § Gate files, the `AGENTS.md` and `.agents/rules-index.md` gate-files rows, and `README.md:260` name the comment and the label (#319). Acceptance: `ateam doctor --check` row counts unchanged; `task spec:check` passes. Evidence: one row edited in each of `AGENTS.md` and `.agents/rules-index.md`, no row added.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#319).
