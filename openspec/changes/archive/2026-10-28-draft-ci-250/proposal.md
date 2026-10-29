# Change Proposal: `draft-ci-250`

Approved-by: Coby Leuschke (2026-10-28)

Amended 2026-10-28 by the owner: narrowed to the draft deferral alone. The merge-check split first ruled here as D5 is parked as #255 (see `design.md` D5).

## Why

Both forges' `CI` and `Gate files` workflows run on every draft push (#250). In the latest 40-workflow-run sample (21 CI runs), all four failed CI runs had a passing `Verify`. Three of them were drafts or near-drafts failing only because flake #248 was open, and `Gate files` turned drafts red before the owner label could apply. PR #247 spent 6m04s on `Verify` as a draft and 6m06s again at the same head once it was ready. A draft is local work with a remote backup. Nothing needs to run on it on either forge until it is marked ready.

## What Changes

- `CI` and `Gate files` (GitHub only; Gitea is unchanged, see below): every job skips when the pull request is a draft, except `Gate files` on `unlabeled`, and on a draft the job is **renamed** (`name: ${{ github.event.pull_request.draft && '<job> (draft, skipped)' || '<job>' }}`) so no check run named `Required` or `Gate files` exists at a draft head. GitHub counts a skipped job as a passing required check, but a required context with no check run is "Waiting for status to be reported" and blocks the merge. The ready run is the first thing to create the required contexts, so marking a draft ready at the same SHA has no stale-pass window (task 1.2 measured 129 s with a plain skip).
- `Required` keeps `if: always()`, written `always() && <not draft>`, and still fails on any needed result other than `success` whenever it runs.
- Gitea: no change. Codex's review of PR #252 showed that Gitea 1.22 cannot defer drafts safely with these workflows. Gitea does not evaluate `name:`. It does not treat `always() && (...)` as a dependency override, so a failed prerequisite would leave `Required` skipped, and Gitea reports a skipped job as success. It emits `label_updated`, not `unlabeled`. And an older WIP run can finish after the ready run starts. The Gitea twins stay byte-identical to `main`, and the twin test maps the GitHub draft lines back to them.
- Workflow tests in `internal/bootstrap` and the "Merge gate and CI workflow enforcement" requirement state this contract; mutation checks show that dropping the guard, the rename, or the skipped-prerequisite rejection is caught.
- `.agents/protocol.md` § 2 Verify says CI on a draft is deferred, not passed, and local gates are unchanged.

## Track

`design`. It touches gate files (`.github/workflows/`, `.gitea/workflows/`) and the CI merge-gate requirement. See `design.md`.

## Non-goals

- No change to `scripts/merge-check.sh`, `scripts/gate-files.sh`, the ruleset, required-check names, the `Required` aggregate's shape, review sequencing (#192), or local `task verify` obligations.
- No merge-check workflow split (parked as its own issue), no `[skip ci]` convention, no new external automation, and no rewrite of the #87 concurrency rule.

## Definition of Done

- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
- Static workflow tests cover the draft guard, the draft rename, the `always()` form on `Required`, the ready trigger, and `Required`'s rejection of skipped jobs, each with a `detection` mutation.
- Recorded live GitHub evidence on a throwaway PR in this repository: a draft head shows no `Required` or `Gate files` check run; after `gh pr ready` at the same SHA the required contexts read "Waiting for status" until the ready run reports; ready synchronize, ready → draft → ready, and auto-merge armed immediately after ready are recorded with their check states. Gitea parity is recorded as executed or as unavailable, with the reason.
- The owner adds `gate-change-approved` before landing.
