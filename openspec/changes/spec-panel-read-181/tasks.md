## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing this hold in the same edit (#181). Hold: OPEN-Q proposal awaits owner approval

## 2. Read view

- [ ] 2.1 Have `/api/openspec` list the in-flight changes from every worktree, each with its `branch`, `worktree` and `proposal`.
  - Premise: `handleOpenSpec` is at `internal/web/server.go:650-670`. `ScanOpenSpecQueue(root)` (`internal/state/openspec.go:306`) reads one root. Worktrees are listed by `parseWorktrees(runGitOutput(ctx, root, "worktree", "list", "--porcelain"))`, as `internal/state/forge.go:295` does.
  - Acceptance: a `server_test.go` case carries `// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access`. It builds a repository with a second worktree holding a change on `OPEN-Q`, and asserts that the change appears once with that worktree's branch, the exact proposal bytes, and `approval.state` taken from the file. A worktree whose change directory cannot be read is skipped and reported, and the response is still returned. `task mutate:check` gives `detection` for scanning only `root`.
- [ ] 2.2 Add the spec panel to the OpenSpec tab: the proposal as preformatted text, task progress, holds with markers, `Approved-by` with its approval state, and the PR link. `OPEN-Q` changes sort first.
  - Premise: the queue panel is at `ui/src/routes/+page.svelte:860-900`. `OpenSpecChange` is at `ui/src/lib/board.ts:79`.
  - Acceptance: a Vitest test on a pure sort-and-link helper in `ui/src/lib/board.ts` checks that `OPEN-Q` sorts first, that ties keep name order, and that a PR link appears only when a forge card matches the branch. `task ui:check` is clean.

## 3. Close

- [ ] 3.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok` naming `node_modules/.bin/openspec`.
