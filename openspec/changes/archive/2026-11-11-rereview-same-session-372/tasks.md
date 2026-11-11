## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit, and rules on design Open Questions 1 (D1) and 2 (D4) (#372).
  - Evidence: the owner approved in Enjoy on 2026-11-11 and asked Claude to write the header. Rulings: D1 option A (the notify text carries the reviewer session); D4 option A (the reviewer declines with `notify --decline`).

## 2. Re-review routing

- [x] 2.1 Carry the reviewer's session in the notify text, from `detectHost` or `--from`, validated as one line of `[A-Za-z0-9._:-]`.
  - Premise: `notifyText` at `cmd/ateam/reviewlaunch.go:273`, `reviewNotify` at `:280`, `detectHost` at `:55`.
  - Acceptance: a `review_test.go` case (`// spec: openspec/specs/model-review/spec.md#Requirement: Review notify text`) checks the exact message with and without a session, and checks that a session holding a newline or shell text exits 2 and sends nothing. The hostile-record scenario still passes.
  - Evidence: `notifyText(pr, sha, session)` and `--from` in `cmd/ateam/reviewlaunch.go`, with the session checked against `sessionID`. `TestReviewNotifyReviewerSession` covers `--from`, the herdr pane from the environment, no session, a newline session and a shell-text session (both exit 2 and send nothing). `TestReviewNotify` (the hostile record) is unchanged and passes.
- [x] 2.2 Add `--reviewer-session` and `--fresh` to `ateam review request`. Re-route through `[notify]` only for a cached `REQUEST CHANGES` record at a non-head sha whose run `chooseReviewer` accepts. Otherwise print the fresh launch with the `rereview` reason. When the notify command errors, print the launch spec instead.
  - Premise: `reviewRequest` at `cmd/ateam/reviewlaunch.go:146`, `chooseReviewer` at `:248`, the cached record at `internal/state/state.go:588`, and the `[notify]` actions in `.agents/hosts/{enjoy,herdr,t3code}.toml`.
  - Acceptance: table cases (`// spec: …#Requirement: Review request spec`) cover a re-route, `APPROVE`, a record at the head, no record, a reviewer the gate now rejects, `--fresh`, and a failing notify runner. `task mutate:check` gives `detection` for dropping the verdict check and for dropping the `chooseReviewer` call.
  - Evidence: `rereview` and `cachedRecord` in `cmd/ateam/reviewlaunch.go` reuse `sendNotify`/`notifyTool`, the helpers taken out of `reviewNotify`. `TestReviewRequestRereview` covers same-session, approve, at-head (abbreviated sha), no record, a workhorse reviewer the gate rejects against an opus implementer, `--fresh`, unreachable (the notify command errors, then the launch runs), and a hostile `--reviewer-session`. `task mutate:check` verdicts are `detection` for removing the verdict case and for ignoring the `chooseReviewer` error, and the source SHA-256 was unchanged after both runs.
  - Review fix (Codex REQUEST CHANGES at `7314919`): the printed `request --reviewer-session` and `--fresh` commands left out the implementer's `--return-to`, so on Enjoy they exited 2. Also, `return_to` entered the reviewer's prompt checked only for newlines. Now the notify, decline and fallback texts all carry `--return-to <id>`, the re-review asks the reviewer to notify with `--from` again, and `return_to` must match the session grammar wherever it enters a message. `TestRereviewPrintedCommandsRun` runs the printed commands on a tool-only host with no session id in the environment. Hostile cases cover a space, `;`, `$(...)` and the 128/129 boundary. `task mutate:check` gives `detection` for dropping either `return_to` check.
- [x] 2.3 Implement the D4 ruling (`notify --decline`, or the round cap).
  - Acceptance: one test on the fixed decline text, or on the cap boundary.
  - Evidence: D4 option A. `notify --decline` sends `declineText` and refuses to be combined with `--record` (exit 2). Covered by the `decline` and `decline-and-record` cases.

## 3. Docs

- [x] 3.1 Update `.agents/skills/review-pr/SKILL.md:35` (pass `--from` on Enjoy, and the re-review paragraph) and `docs/daily-use.md:84`.
  - Acceptance: `ateam doctor --check` parity is green.
  - Evidence: the `review-pr` skill and its `ateam init` template copy (`internal/bootstrap/templates.go`), `docs/daily-use.md` step 11, `docs/adoption.md` (the notify text), and the `ateam review` usage line are updated. `ateam doctor --check` reports `violations: null`.

## 4. Close

- [x] 4.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok`.
  - Evidence: archived with `node_modules/.bin/openspec` 1.14.3 (`+ 1 added, ~ 1 modified` in `openspec/specs/model-review/spec.md`); `task verify` exit 0 on a clean tree, with `openspec: strict ok` naming `node_modules/.bin/openspec`.
