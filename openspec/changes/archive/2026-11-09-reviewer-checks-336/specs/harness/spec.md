## ADDED Requirements

### Requirement: Reviewer checks policy key

`.agents/review-policy.json` SHALL name the checks a reviewer must run and pass at the current head under `reviewer_checks`. It SHALL accept `required_checks` as a deprecated alias with the same meaning for one release, SHALL reject a policy that sets both keys, and `ateam doctor` SHALL warn when the deprecated key is present. `reviewer_checks` is unrelated to `ATEAM_REQUIRED_CHECKS`, which names the forge status contexts the merge gate requires.

#### Scenario: New key and deprecated alias parse alike

- **WHEN** a policy with models sets `reviewer_checks: ["task verify"]`, or instead sets only `required_checks: ["task verify"]`
- **THEN** `ParsePolicy` returns the same `ReviewerChecks` for both, and a review record lacking reviewer-sourced passing evidence for `task verify` at the current head is rejected with `reviewer check "task verify" lacks reviewer-executed passing evidence at current head`

#### Scenario: Both keys are rejected

- **WHEN** a policy sets both `reviewer_checks` and `required_checks`, even when either one is an empty array
- **THEN** `ParsePolicy` fails with an error naming both keys

#### Scenario: Doctor names the replacement

- **WHEN** `ateam doctor` runs in a repository whose `.agents/review-policy.json` sets `required_checks`
- **THEN** it warns `.agents/review-policy.json: required_checks is deprecated; rename it to reviewer_checks`, and it does not warn for a policy using `reviewer_checks` or setting neither key
