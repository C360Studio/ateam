# Change Proposal: `draft-ci-250`

Approved-by: Coby Leuschke (2026-10-28)

## Why

Both forges' `CI` and `Gate files` workflows run on every draft push (#250). In the latest 40-workflow-run sample (21 CI runs), all four failed CI runs had a passing `Verify`. Three of them were drafts or near-drafts failing only because flake #248 was open, and `Gate files` turned drafts red before the owner label could apply. PR #247 spent 6m04s on `Verify` as a draft and 6m06s again at the same head once it was ready. Drafts are checkpoints. Running the full gate on them uses runner time and reports "not ready to merge" as a broken build.

## What Changes

- `CI` (both forges): `Verify` and `Required` (now `needs: [verify]` only) skip when the pull request is a draft (D2 = A, grey). They run in full on non-draft `opened`/`reopened`, on `ready_for_review`, on every `synchronize` while the pull request is ready, and on push to `main`.
- `Merge check` (both forges) moves to its own workflow and required context, `merge-check.yml`, triggered on `opened`, `synchronize`, `reopened`, `ready_for_review`, `edited`, `labeled`, `unlabeled`; it skips drafts and keeps the base-branch checkout (#140). It measures PR state, not code, so it reruns on the events that change that state: the reviewer's `reviewed-by:` body edit clears it without a manual rerun (D5).
- `Gate files` (both forges): the job skips on draft events and adds `edited`. It runs on ready, on later pushes, on title/body edits, and on label changes to a ready pull request, still from the trusted base.
- `Required` still rejects a skipped, failed, cancelled, or missing prerequisite whenever it runs. GitHub treats a skipped required check as passing, so a draft head marked ready at the same SHA carries a skipped `Required` until the ready run's check run exists. Task 1.2 measures that window before the workflow lands; `design.md` D2 carries the decision rule.
- The `main` ruleset requires `Required`, `Merge check`, and `Gate files` (owner edit). `review-pr` § 5 drops the manual rerun and `.agents/protocol.md` § 4 drops the "expect one red Merge check" sentence (#192).
- Workflow tests in `internal/bootstrap` and the "Merge gate and CI workflow enforcement" requirement state this scheduling contract. Mutation checks show that dropping the draft guard, the ready trigger, or the skipped-prerequisite rejection is caught.
- `.agents/protocol.md` § Verify says CI on a draft is deferred, not passed, and that local gates are unchanged.

## Track

`design`. It touches gate files (`.github/workflows/`, `.gitea/workflows/`) and the CI merge-gate requirement. See `design.md`.

## Non-goals

- No change to `scripts/merge-check.sh`, `scripts/gate-files.sh`, the review-record format, the `Required` and `Gate files` context names, review-after-ready sequencing, or local `task verify` obligations. The ruleset edit adding `Merge check` is the owner's.
- No `[skip ci]` convention, no new external automation, and no rewrite of the #87 concurrency rule beyond what the draft guard needs.

## Definition of Done

- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
- Static workflow tests cover the draft guard, the ready trigger, and `Required`'s rejection of skipped jobs, and each has a `detection` mutation.
- Recorded live GitHub evidence on a throwaway PR in this repository for: draft opened, draft synchronize, draft → ready at the same SHA, ready synchronize, ready → draft → ready, and auto-merge armed immediately after ready. The record states the check states that `Required` and `Gate files` showed at each step. Gitea parity is recorded as executed or as unavailable, with the reason.
- The owner adds `gate-change-approved` before landing.
