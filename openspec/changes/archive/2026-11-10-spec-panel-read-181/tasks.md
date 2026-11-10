## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit (#181).
  - Evidence: the owner approved in Enjoy on 2026-11-10 and asked Claude to write the header.

## 2. Read view

- [x] 2.1 Have `/api/openspec` list the in-flight changes from every worktree, each with its `branch`, `worktree` and `proposal`.
  - Premise: `handleOpenSpec` is at `internal/web/server.go:650-670`. `ScanOpenSpecQueue(root)` (`internal/state/openspec.go:306`) reads one root. Worktrees are listed by `parseWorktrees(runGitOutput(ctx, root, "worktree", "list", "--porcelain"))`, as `internal/state/forge.go:295` does.
  - Acceptance: a `server_test.go` case carries `// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access`. It builds a repository with a second worktree holding a change on `OPEN-Q`, and asserts that the change appears once with that worktree's branch, the exact proposal bytes, and `approval.state` taken from the file. A worktree whose change directory cannot be read is skipped and reported, and the response is still returned. `task mutate:check` gives `detection` for scanning only `root`.
  - Evidence: `state.ScanWorktreeChanges` (`internal/state/openspec.go`) lists every worktree on a branch, leaves out detached review checkouts, and names unreadable worktrees in `worktree_skipped`. `TestScanWorktreeChanges` covers a held change, a stale digest of zeros (independent oracle: no proposal hashes to it), a detached checkout of the held branch, and a worktree whose `tasks.md` is a directory. `TestOpenSpecAPIListsWorktreeChanges` checks the endpoint over a root with no change. `task mutate:check` verdicts are `detection` for scanning `root` instead of each worktree and for dropping the detached-worktree skip.
- [x] 2.2 Add the spec panel to the OpenSpec tab: the proposal as preformatted text, task progress, holds with markers, `Approved-by` with its approval state, and the PR link. `OPEN-Q` changes sort first.
  - Premise: the queue panel is at `ui/src/routes/+page.svelte:860-900`. `OpenSpecChange` is at `ui/src/lib/board.ts:79`.
  - Acceptance: a Vitest test on a pure sort-and-link helper in `ui/src/lib/board.ts` checks that `OPEN-Q` sorts first, that ties keep name order, and that a PR link appears only when a forge card matches the branch. `task ui:check` is clean.
  - Evidence: `specPanelRows` in `ui/src/lib/board.ts` has two Vitest cases (sort order with name and branch ties; PR link prefers the unmerged card and is absent when only an issue card has the branch). The panel loads `/api/openspec` when the OpenSpec tab opens, sits at the top of the tab (below 98 requirements it was about 10,000 px down), and was checked in the browser against this repository: `spec-approve-181` (`OPEN-Q`, unapproved) first and open, then `review-lease-314` (unbound) and `spec-panel-read-181` (approved).
  - Review fix (Codex REQUEST CHANGES at `f36b5f2`): the panel kept repository A's rows after a switch to B, and a late or older response could overwrite newer state. `createSpecPanel` (`ui/src/lib/dashboard.ts`) follows the dashboard loader's generation guard: `changeRepository` resets it and reloads when the OpenSpec tab is open, and only the latest load publishes rows, loading or an error. Deferred-response Vitest cases cover a completed A load cleared on switch, late A success and failure after B, and an older refresh completing last. Removing the guard fails 3 of them (`cp` + SHA-256 restore verified).

## 3. Close

- [x] 3.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok` naming `node_modules/.bin/openspec`.
