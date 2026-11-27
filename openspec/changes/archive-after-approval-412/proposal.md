# Proposal: archive-after-approval-412
Approved-by: Coby Leuschke (2026-11-27) sha256:648da44c7e60

## Track

`quick`: wording in the Close rule, its repo copies and the templates `ateam init` ships; no spec Requirement, capability, gate file, CLI flag or boot path changes. The GitHub merge gate already accepts the commit this adds, so no code changes.

## Decisions you are making by approving

- **On GitHub, a PR that lands an OpenSpec change is reviewed unarchived.** It is marked ready with the change still under `openspec/changes/<id>/`; a `REQUEST CHANGES` round appends its tasks there. Once the latest record approves, the developer moves the change to `openspec/changes/archive/<date>-<id>/` in one commit and only then arms auto-merge. No more unarchive and re-archive per round.
- **The PR no longer merges on its own after approval.** Today auto-merge is armed at ready, so the reviewer's `reviewed-by:` line lets it fire. Under this rule someone must archive and arm after the approval: under `/coordinate` that is the developer run the coordinator already wakes on `APPROVE`; under `review_launch: manual` it is you or a new developer session.
- **The archive commit needs no new review on GitHub**, because `task merge:check` passes a record followed only by document commits (`gateIsDoc`, `internal/review/gate.go:727`; `gateCommitAllowed`, `gate.go:763`). An archive commit that touches anything else is no longer document-only and needs a new review, as today.
- **On Gitea nothing changes:** archive before ready and arm at ready, because the Gitea gate wants a new review after every later commit (`gate.go:664`, owner ruling 5 on #143). Teaching it to pass an archive commit needs a live proof that Gitea's commit file lists are complete and a gate-file change; not here.
- **The reviewer no longer reads the archived tree.** The canonical `openspec/specs/<cap>/spec.md` that `openspec archive` merges from the reviewed delta lands unreviewed, and a strict-validation failure that only shows after the merge (a MODIFIED Requirement over 500 characters) turns CI red after approval. Its fix is document-only and needs no new review; `spec-check`'s delta-length warning (#398) catches most before review.
- **The change's last task is the archive and stays open during review**, so `spec:queue` and `ateam doctor` show the change in flight, not "done, not archived", while it waits.

## Why

The owner ruled on #412 (2026-11-24) to archive once the PR is approved, before auto-merge is armed, with the caveat that the archive commit must not trigger another review. On semsource#365 every `REQUEST CHANGES` round cost an unarchive, new tasks and a re-archive.

## What Changes

1. Protocol § 2 step 5 (Close) and `protocolCloseStep`: the order above, the Gitea exception, and that a PR with no OpenSpec change arms at ready as today. "Merging with a completed unarchived change is a Close-gate miss" stays.
2. `AGENTS.md` Close gate bullet and Close rule row, `.agents/rules-index.md` row, and `closeRuleRow`: the same order in one line each.
3. `docs/daily-use.md` § 5 step 4 and § 6 steps 1–2: archive after approval, then arm.
4. A bootstrap test asserts the new order and the Gitea exception in the generated protocol and Close row.

## Scope & Non-Goals

- **In-bounds:** `.agents/protocol.md`, `AGENTS.md`, `.agents/rules-index.md`, `internal/bootstrap/templates.go`, their bootstrap tests, `docs/daily-use.md`.
- **Non-goals:** `internal/review/` and every other gate file; the Gitea gate's later-commit rule; the `review_record: aip` path, whose record already lives in the archived change (`reviewrecord.go:29`); `ateam/review-record` status after the archive commit, which #414 (PR #435) fixes and which shows "behind head" until then, informational only; the coordinate skill and coordinator contract, which already route `APPROVE` to the developer run to close per § 2 step 5; the issue's second option, appending to an archived change.

## Definition of Done

- The repo copies and the generated files carry the same Close rule (the existing parity test, `parity_test.go:162`), the new test fails when the archive-after-approval sentence or the Gitea exception is removed (`task mutate:check` or `cp` + SHA-256), and `merge_method` still rewrites all three `squash` spots.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; this change is archived under the rule live when the PR is approved.
