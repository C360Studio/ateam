# Change Proposal: `reviewer-record-262`

Approved-by: Coby Leuschke (2026-10-30)

## Why

A reviewer working only from its role contract can approve and still leave `Merge check` red. The base reviewer contract's § Record (`.agents/contracts/reviewer.md:11-22`, template `internal/bootstrap/review.go:41-55`) never mentions the PR-description `reviewed-by:` line that the gate needs (`internal/review/gate.go:431`). That rule lives only in `review-pr` § 5. This happened on C360Studio/semteams#387 (#262).

Two related gaps belong to the same path. First, an approval followed by a rejection at the same head is only a comment, and `merge-check.yml` does not run on `issue_comment`, so the earlier green stays until someone reruns the check. Second, by owner ruling on 2026-10-30, the run before a review record exists should not count as a failure. Every code PR currently posts one red `Merge check` when marked ready, then one more per fix push and per description edit, each with the finding `no line starts reviewed-by:` and a failure email (three on #267 in one morning).

## What Changes

- Both `merge-check.yml` workflows skip the `merge-check` job, under the non-required name `Merge check (awaiting review record, skipped)`, until the PR description contains `reviewed-by:`. This reuses the #250 draft-skip mechanism. Once the line is present, the full gate runs unchanged from the base commit.
- The reviewer contract's § Record (template and this repo's copy) names the description line. It tells the reviewer to refresh and verify `Merge check` after either verdict, and to rerun the whole workflow when a rejection follows an approval at the same head.
- `review-pr` § 5 (skill and template), protocol § 3 review, and `docs/daily-use.md` replace the "one red `Merge check`" promise with the skipped state and the rerun-after-rejection path.
- The harness spec's "Merge check runs on pull-request state events" requirement gains the awaiting-record scenario and the rejection-after-approval scenario. Its pin scenario names the new guard.
- `TestMergeCheckWorkflowPinned` and its sensitivity table pin the new guard and name.

## Track

`design`: this touches gate files (`.github/workflows/`, `.gitea/workflows/`). It changes a harness spec Requirement, and it changes adopter-visible contract text, which `ateam doctor` drift-checks (`internal/bootstrap/review.go:143`). See `design.md`. The PR needs the owner's `gate-change-approved` label.

## Non-goals

- No change to what `ateam review gate` or `scripts/merge-check.sh` judge, to the review-record grammar, or to `ATEAM_REQUIRED_CHECKS`.
- No `issue_comment` trigger or other comment-driven automation, and no privileged workflow (#192, the #262 acceptance).
- No new `ateam doctor` check for the sentence. Parity already flags a reviewer contract that differs from the template.
- No change to the `review_record: aip` contract variant (`internal/bootstrap/reviewrecord.go:59-63`), which has no PR.

## Definition of Done

- On a ready PR with no `reviewed-by:` line, `Merge check` is skipped under the awaiting name and no failed run is posted. The approving description edit runs the full gate, and it passes. Both are recorded on this PR with run IDs.
- A rejection after an approval at the same head is followed by a whole-workflow rerun that is observed failing. This is recorded on this PR if the owner authorizes it (design Open Question 3), and otherwise named as unproven.
- `TestMergeCheckWorkflowPinned` and its sensitivity rows pass, with each new mutant (guard dropped, name kept while awaiting) reported.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green on a clean tree.
