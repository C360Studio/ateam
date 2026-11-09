# Change Proposal: `reviewer-checks-336`

Approved-by:

## Why

Two different things are both called "required checks", and adopters and agents mix them up (#336). The `required_checks` key in `.agents/review-policy.json` lists checks the *reviewer* must run and pass at the current head (`internal/review/policy.go:17`, enforced at `internal/review/record.go:227`). `ATEAM_REQUIRED_CHECKS`, by contrast, lists the forge status contexts the merge gate needs the `main` ruleset to require (`cmd/ateam/review.go:190`, set in both `merge-check.yml` workflows).

## What Changes

- Rename the policy key to `reviewer_checks` (`Policy.ReviewerChecks`).
- `required_checks` keeps parsing as a deprecated alias for one release, because the strict decoder (`internal/review/json.go:84`) would otherwise reject existing adopter policies. A policy that sets both keys is rejected.
- `ateam doctor` warns when the deprecated key is present and names the replacement.
- Error text, `docs/model-review.md:38`, and the test fixtures use the new name. One sentence in `docs/model-review.md` separates it from `ATEAM_REQUIRED_CHECKS`.
- One new spec Requirement covering the key, the alias, and the both-keys rejection.

## Track

`design`: this renames exported surface reachable from adopting repositories, adds a spec Requirement, and touches a gate file (`internal/review/`). See `design.md`.

## Non-goals

- Renaming `ATEAM_REQUIRED_CHECKS` or changing what the merge gate requires.
- Removing the alias. That waits for a later release (design Open Question 2).
- Editing ADR 0002, which records the key's history under its old name.
- Enabling the dormant `review route` / `review check` path.

## Definition of Done

- A policy with `reviewer_checks`, or with only `required_checks`, parses to the same `ReviewerChecks`. A policy with both is rejected with an error naming both keys.
- `ateam doctor` warns on `required_checks` and stays quiet on `reviewer_checks` or a policy without either key.
- `task mutate:check` reports `detection` on the alias fold, the both-keys rejection, and the doctor warning.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green. The change is archived before ready, and the owner approves the gate change at the final head.
