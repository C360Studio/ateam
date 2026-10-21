# Change Proposal: `ci-required-cancelled`

## Why

Marking PR #84 ready for review started a second `pull_request` run on the same head SHA. The workflow's `concurrency.cancel-in-progress` cancelled the run already in flight; its `Verify` and `Merge check` jobs ended `cancelled`, and `Required`, which runs under `if: always()` so that a failed or cancelled need can never leave it skipped, correctly reported `failure`. Both a failed and a passing `Required` then sat on one SHA and the `main` ruleset blocked the merge until the cancelled run was rerun to green (#87). This recurs on every draft-to-ready transition with an unchanged head.

Switching `Required` to `if: ${{ !cancelled() }}` was considered and rejected: a skipped required check satisfies the ruleset, so a manually cancelled run would leave a mergeable PR with no verification. The defect is in what gets cancelled, not in how `Required` reports.

## What Changes

1. **Workflow (`.github/workflows/ci.yml`):** `concurrency.cancel-in-progress` becomes `${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}`. Superseded pushes are still cancelled; the `ready_for_review` run queues behind the in-flight run for the same head and both complete.
2. **Pinned-workflow test (`internal/bootstrap/mergecheck_test.go`):** `ciWorkflowViolations` reads the `concurrency` block and rejects any `cancel-in-progress` other than that expression or `false`; the clean fixture carries the block and two sensitivity cases plant the old expression and an unconditional `true`.
3. **Contract (`openspec/specs/harness/spec.md`):** the "Pinned CI workflow and Required aggregate" scenario names the required `cancel-in-progress` expression and why.
