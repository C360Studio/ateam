# Proposal: rereview-after-approve-452
Approved-by: Coby Leuschke (2026-11-28) sha256:302436cccf1a

## Track

`quick`: it modifies two existing Requirements in one capability (`model-review`) and adds none, adds no exported surface (the `--reviewer-session` flag already exists), and is no boot path, durability/ack behavior or gate file. The code change is `cmd/ateam/reviewlaunch.go` only; `internal/review/`, the host manifests and the merge-check script are untouched, so `Gate files` stays green and no `gate-change-approved` is needed.

## Decisions you are making by approving

- **An approved PR whose head moved can go back to the same reviewer.** Today `ateam review request --reviewer-session <id> <pr>` sends the new commits back to the reviewing session only after `REQUEST CHANGES`; after `APPROVE` it prints a fresh full-review launch (`rereview: record verdict is approve`). After this change an `APPROVE` record at an older commit is treated like `REQUEST CHANGES`: the same session is asked to review `<record-sha>..<head>` and post a new record, under the same checks as today (the reviewer still passes the gate comparison, the host can reach it, the record is not at the head). `--fresh` still forces a new reviewer.
- **The message says which verdict it follows.** After `APPROVE` it reads `after your APPROVE at <sha>` and drops "confirm each finding of your record is closed" (an approving record has no blocking finding to close; non-blocking ones go to follow-up issues). After `REQUEST CHANGES` the text is unchanged.
- **The merge gate does not change.** Which `main` merges keep an approving record (merges touching none of the PR's files, and docs-only commits) stays as it is; every catch-up that touches the PR's files still needs a new record. This only makes that record cheap: one same-session read of the new commits instead of a full review by a new reviewer.

## Why

#452, with v0.4.8 field evidence: after each sibling merge, PRs #483, #487 and #488 were approved, caught up with `main` through a merge touching `openspec/specs/harness/spec.md` (or the archive commit), `merge:check` refused with `review record ... is older than content commit`, and `ateam review request --reviewer-session` printed a fresh full-review launch, costing a full review per catch-up. #411 worked around the refusal by messaging the reviewer directly.

## What Changes

1. `rereview` in `cmd/ateam/reviewlaunch.go` accepts `APPROVE` as well as `REQUEST CHANGES`; `rereviewText` takes the verdict.
2. `TestReviewRequestRereview`: the `approve` case becomes a same-session send with the APPROVE wording; a missing verdict still falls back.
3. `docs/daily-use.md` step 11 names the catch-up case.
4. `model-review` spec: *Re-review to the reviewing session* and *Review blocking bar and round scope* modified.

## Scope & Non-Goals

- **In-bounds:** `cmd/ateam/reviewlaunch.go`, `cmd/ateam/reviewlaunch_test.go`, `docs/daily-use.md`, the spec delta.
- **Non-goals:** widening which `main` merges keep a record (weakens a gate; not proposed); the notify text's "to re-review after fixes" wording (a fixed-text Requirement; it already carries the line after `APPROVE`); the reviewer contract, `review-pr` and `coordinate` skills (the re-review message carries its own instructions, and the coordinator already runs this command after a catch-up); detecting whether a catch-up actually needs a record (`merge:check` decides that).

## Definition of Done

- The test shows an `APPROVE` record at an older sha sends the APPROVE re-review text to the reviewer session and launches nothing, and a record with no verdict still prints a fresh launch; a non-destructive mutation restoring the old refusal is a `detection`.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
