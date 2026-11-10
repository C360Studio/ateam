## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing this hold in the same edit, ruling on design OQ1 (name override) and OQ2 (GitHub commit count) (#181).
  - Evidence: the owner approved in Enjoy on 2026-11-10 and asked Claude to write the header. OQ1: login only, with no name field in the request. OQ2: two commits through the contents API on GitHub, `proposal.md` first.
- [x] 1.2 `spec-panel-read-181` has merged; the button sits in its panel.
  - Evidence: PR #367 squash-merged as `5ba5561` on 2026-11-10, and `main` is merged into this branch.

## 2. Forge file helper

- [ ] 2.1 Add a forge helper in `internal/state` that resolves a branch head, reads files at that commit with their blob `sha`, writes contents with the `sha` (Gitea: one multi-file commit; GitHub: one `PUT` per file in the order given), and returns the login behind `WriteToken()` (D1 A, D2, D3).
  - Premise: dispatch as in `PublishCommitStatuses` (`internal/state/commitstatus.go:338-360`); GitHub through `ghCommand(ctx, WriteToken(), ...)` (`:449-463`); Gitea through `net/http` (`:470`).
  - Acceptance: tests with recorded GitHub and Gitea responses carry `// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board`. They assert the same things on both providers: reads use the head commit SHA, not the branch name; writes send the read blob `sha`; a stale `sha` returns the conflict error; and the GitHub write order is `proposal.md` then `tasks.md`. `task mutate:check` gives `detection` for swapping the write order and for reading at the branch name.

## 3. Endpoint and button

- [ ] 3.1 Add `POST /api/openspec/approve` `{change, branch}` behind `validateMutationRequest` (`internal/web/server.go:155`), applying `ApproveEdits` (`internal/state/approval.go:144`) to the fetched bytes, with the D4 shortcut comment and commit message.
  - Acceptance: server tests with a fake helper check that an `OPEN-Q` change writes two files whose header is the bound form for the fetched `proposal.md` and `design.md`; that a stale `sha` returns 409; that a missing hold returns 400; that `branch: main` returns 400; and that a cross-origin request returns 403. Removing the hold check gives `detection`.
- [ ] 3.2 Add the Approve button to the spec panel, only for a change holding on `OPEN-Q`, showing the error text on 409 or 502.
  - Acceptance: a Vitest test checks that the button is absent on an approved change and present on an `OPEN-Q` one.

## 4. Live check and close

- [ ] 4.1 The owner approves one change from the board on this repository, then `task spec:check` on that branch passes. Record the one-line result here.
- [ ] 4.2 Archive before ready with `task verify` green and `openspec: strict ok` naming `node_modules/.bin/openspec`.
