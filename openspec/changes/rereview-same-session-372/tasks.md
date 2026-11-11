## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit, and rules on design Open Questions 1 (D1) and 2 (D4) (#372).
  - Hold: OPEN-Q proposal awaits owner approval

## 2. Re-review routing

- [ ] 2.1 Carry the reviewer's session in the notify text, from `detectHost` or `--from`, validated as one line of `[A-Za-z0-9._:-]`.
  - Premise: `notifyText` at `cmd/ateam/reviewlaunch.go:273`, `reviewNotify` at `:280`, `detectHost` at `:55`.
  - Acceptance: a `review_test.go` case (`// spec: openspec/specs/model-review/spec.md#Requirement: Review notify text`) checks the exact message with and without a session, and checks that a session holding a newline or shell text exits 2 and sends nothing. The hostile-record scenario still passes.
- [ ] 2.2 Add `--reviewer-session` and `--fresh` to `ateam review request`. Re-route through `[notify]` only for a cached `REQUEST CHANGES` record at a non-head sha whose run `chooseReviewer` accepts. Otherwise print the fresh launch with the `rereview` reason. When the notify command errors, print the launch spec instead.
  - Premise: `reviewRequest` at `cmd/ateam/reviewlaunch.go:146`, `chooseReviewer` at `:248`, the cached record at `internal/state/state.go:588`, and the `[notify]` actions in `.agents/hosts/{enjoy,herdr,t3code}.toml`.
  - Acceptance: table cases (`// spec: …#Requirement: Review request spec`) cover a re-route, `APPROVE`, a record at the head, no record, a reviewer the gate now rejects, `--fresh`, and a failing notify runner. `task mutate:check` gives `detection` for dropping the verdict check and for dropping the `chooseReviewer` call.
- [ ] 2.3 Implement the D4 ruling (`notify --decline`, or the round cap).
  - Acceptance: one test on the fixed decline text, or on the cap boundary.

## 3. Docs

- [ ] 3.1 Update `.agents/skills/review-pr/SKILL.md:35` (pass `--from` on Enjoy, and the re-review paragraph) and `docs/daily-use.md:84`.
  - Acceptance: `ateam doctor --check` parity is green.

## 4. Close

- [ ] 4.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok`.
