# Change Proposal: `review-strength-82`

## Why

merge-check requires a model on both review lines (#108), but it doesn't compare them, so "a reviewer is never weaker than the implementer" is still unenforced. ADR 0002 settles the rule: rank each run by model tier, then by effort band, and pass a reviewer that is not weaker than any implementer. It also makes the cross-agent requirement optional (#82, ADR 0002 § Implementation item 2).

## What Changes

- `ateam review compare --implementer <agent:model:effort>... --reviewer <agent:model:effort>` applies ADR 0002 rule 2 to `.agents/review-policy.json`. It exits 0 on a pass, 1 on a finding, and 2 when it can't decide. With no policy file it prints `skipped: no policy` and passes.
- `scripts/merge-check.sh` passes the runs from both PR-body lines to `ateam review compare` (the binary named by `ATEAM`, default `ateam`). Once a policy exists, effort is required on both lines. The different-agent check applies only when the policy sets `cross_provider: true`.
- The CI merge-check job and `task merge:check` build the binary they call.
- The instruction text listed in ADR 0002 § 5 changes from "a different agent reviews; effort is not rank" to "not weaker by tier and band; cross-provider optional".
