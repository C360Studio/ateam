# Change Proposal: `spec-approve-181`

Approved-by: Coby Leuschke (2026-11-10) sha256:ff62376465c6

## Why

The owner approves a change by editing two files on an agent's branch: filling `Approved-by:` in `proposal.md` and removing the `OPEN-Q` hold in `tasks.md`. Today that means a checkout, an edit, a commit and a push, or asking the agent to write the header for them (`2026-11-09-bounded-board-reads-199/tasks.md` 1.1). #181 asks for one button on the board's spec panel that does that edit as a forge commit. #272 already provides the edit as a pure function, `ApproveEdits` (`internal/state/approval.go:144`), and `spec-panel-read-181` provides the panel that the button sits in.

## What Changes

- A new `POST /api/openspec/approve` with `{change, branch}`, behind `validateMutationRequest` (`internal/web/server.go:155`). It reads `proposal.md`, `design.md` and `tasks.md` from the branch head on the forge and applies `ApproveEdits`. It then writes the two edited files through the forge contents API, using each file's current blob `sha`. The approver's name is the forge login behind the write token.
- A new forge helper in `internal/state` reads files at a ref and writes contents for GitHub and Gitea, dispatched on `ForgeConfig.Provider` like `PublishCommitStatuses` (`internal/state/commitstatus.go:338`). Gitea writes both files in one commit. GitHub writes two commits, `proposal.md` first.
- A stale blob `sha` returns 409. A change with no `OPEN-Q` approval hold returns 400. The panel shows the Approve button only on a change holding on `OPEN-Q`.
- One new spec Requirement covers the endpoint, the commit order, the identity and the trust boundary.

## Track

`design`: it adds a spec Requirement and the first forge file write. See `design.md`.

## Non-goals

- Edit, create and delete of changes stay as follow-ups (#181).
- No new trust boundary for loopback clients: the owner ruled on 2026-11-10 that this endpoint keeps today's loopback trust, and `design.md` records the risk.
- No local filesystem write, no write to `main`, and no change to merge-check or `spec:check`.

## Definition of Done

- Forge helper tests with recorded GitHub and Gitea responses make the same assertions on both providers: the read is at one ref, the write sends the current blob `sha`, a stale `sha` maps to the conflict error, and GitHub writes `proposal.md` first.
- Server tests check that approve on an `OPEN-Q` change calls the helper with the two edited files and the bound header, that a stale hash is 409, that a change with no hold is 400, and that a cross-origin request is 403.
- A UI test checks that the Approve button appears only on a change holding on `OPEN-Q`.
- The owner approves one change from the board on this repository, then runs `task spec:check` on the branch and sees it pass. The one-line result goes in `tasks.md`.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before the PR is marked ready.
