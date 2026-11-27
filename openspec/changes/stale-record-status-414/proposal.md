# Proposal: stale-record-status-414
Approved-by:

## Track

`quick`: one MODIFIED Requirement in the harness capability, code in `internal/state/commitstatus.go` and `cmd/ateam/reviewlaunch.go`, and the docs status table. No new command, flag, or setting, and no gate file: the gate's commit rule is copied, not exported, as `changeDirs` already copies `gateChangeDirs` (`internal/state/commitstatus.go:312-314`).

## Decisions you are making by approving

- When a reviewer finishes and runs the notify command, that command also sets the review-record status on the commit the review names, so the PR page shows the verdict even when auto-merge lands before the author's next sync. The reviewer's checkout needs the same write token a sync uses; without it the reviewer sees one warning and the notify still goes out.
- A review record stays shown as current, with its verdict, when every commit after it is a document-only commit or a clean merge of `main`, the same commits `task merge:check` lets through on GitHub. The archive commit your #412 ruling adds after approval then shows the approval, not "record is behind head". On Gitea, where the gate wants a new review after any later commit, the status stays "behind head" as today.
- A rejecting record carries over the same way: a docs-only commit after `REQUEST CHANGES` still shows the red status.
- Merged pull requests are not swept by sync (issue option 2), and no docs-only fallback (option 3) is added: the notify publish covers the #413 race at its source.

## Why

On PR #413 (2026-11-24) the approving record landed at df3b4fd and auto-merge fired two minutes later. Only the author's sync and handoff publish `ateam/review-record`, and sync skips merged PRs (`internal/state/commitstatus.go:578`), so df3b4fd shows "record at 3ddc894 is behind head" forever next to a passing `Merge check`. #412's archive-after-approval adds a docs-only commit after every approving record, which today always leaves the status pending (`commitstatus.go:172-175`). Hosts such as T3 Code read these statuses as their PR seam (#302).

## What Changes

1. `ateam review notify --record` publishes only `ateam/review-record` on the record's sha, with the state and description `chooseStatuses` gives when that sha is the head. `--decline` publishes nothing. `ATEAM_COMMIT_STATUSES=0` / `commit_statuses: false` turn it off; a failed write warns once and leaves the exit code and the message delivery unchanged.
2. `chooseStatuses` treats a record behind the head as current when the commits between them, read from the local checkout, are all document-only non-merge commits or `main` merges touching none of the PR's files (GitHub only). A commit missing locally keeps today's "behind head".
3. Harness spec: the Requirement names notify; the Review record scenario gains the later-commit rule; one notify scenario.
4. `docs/adoption.md` status table and paragraph name notify and the later-commit rule.

## Scope & Non-Goals

- **In-bounds:** `internal/state/commitstatus.go`, `cmd/ateam/reviewlaunch.go`, their tests, `openspec/specs/harness/spec.md`, `docs/adoption.md`.
- **Non-goals:** `internal/review/` and every other gate file; the `ateam/proposal` and `ateam/handoff` contexts; refreshing merged PRs from sync; the `review-pr` skill text (notify's own warning names a failed write); the Gitea gate's later-commit rule (#412's design).

## Definition of Done

- A notify test against a fake forge sees one `ateam/review-record` write on the record sha (`success` for APPROVE, `failure` for REQUEST CHANGES), none with statuses disabled or on `--decline`, and exit 0 with a warning when the write fails.
- `chooseStatuses` table tests: docs-only commit after the record → verdict status; code commit → "behind head"; clean `main` merge → verdict; a `main` merge touching a PR file → "behind head"; Gitea → "behind head"; missing commit → "behind head". Mutation checks detect dropping the docs-only rule and dropping the merge-file rule.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` green; the change is archived after approval per #412's ruling or before ready, whichever rule is live when the PR is ready.
