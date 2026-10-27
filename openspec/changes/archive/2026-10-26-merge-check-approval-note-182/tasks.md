## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#182). Evidence: approved by the owner in Enjoy on 2026-10-26; Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 Export the `proposal.md` header parser from `internal/state` (#182). Evidence: `parseProposal` renamed `ParseProposal` in `internal/state/openspec.go`; `internal/review` imports it (`internal/state` depends only on `internal/testspec`).
- [x] 2.2 `gateSource.fileAt` on GitHub and Gitea, with the fake-forge fixture answering it (#182). Evidence: both forges read the contents API (`repos/{owner}/{repo}/contents/<path>?ref=<head>`, base64 decoded by `gateContent`), so Gitea reuses the guarded JSON reader instead of the raw endpoint the proposal named; 404 is `nil` content (GitHub: gh's `(HTTP 404)`; Gitea: `forgeStatusError`). Fixture: `gateWorld.proposals` / `proposalsMissing`; the fake gh answers an `.err` file on stderr with exit 1; the fake Gitea refuses a contents read not at head.
- [x] 2.3 Collect touched change directories on a code PR and print the note; read errors are notes, never unavailable (#182). Evidence: `gateChangeDirs` and `approvalNotes` in `internal/review/gate.go`, called after the code-file list, before the review lines.
- [x] 2.4 Tests: unapproved, approved, archived path, missing proposal, read error; mutation checks with `task mutate:check` (#182). Evidence: `TestGateApprovalNote` in `internal/review/gate_test.go` (unapproved, approved, header only inside a fence, archived change with no proposal, two changes each noted once, files outside a change directory, documents-only PR, read failure on GitHub, read failure on Gitea), every case on both forges unless named. Mutants under `/tmp/mut182`, `internal/review/gate*.go` SHA-256 unchanged: call removed, verdict inverted, read-error note silenced, archive prefix not handled, dedupe removed, GitHub 404 not mapped, Gitea 404 not mapped: each `detection`.
- [x] 2.5 Harness spec scenario (#182). Evidence: requirement "Merge check notes an unapproved OpenSpec change" with three scenarios.

Generated checks: not used. The inputs are a file list and one header; the table names each path shape (`changes/<id>/`, `changes/archive/<id>/`, files directly under `changes/` and `archive/`) and each read outcome (content, 404, failure).
