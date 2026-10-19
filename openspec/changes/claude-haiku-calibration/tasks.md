# Tasks

## Implementation
- [x] Review plan independently and inspect installed Claude CLI / official accounting contract.
- [x] Implement API-only isolated Claude execution and native evidence parsing.
- [x] Integrate native configuration, stop rules, replay and CLI self-test dispatch.
- [x] Add A/A and A/B examples and explain the fixed-model workflow.

## Verification
- [x] Prove focused parser/execution/accounting regressions and mutations; record generated-check decision.
- [x] Complete independent change review and clean-tree task verify.
- [x] Publish implementation and evidence through draft PR #13 (implementation checkpoint 23f7cd5; live follow-up accompanies this update).

## Live calibration
- [x] Run and report Haiku A/A with immutable artifacts and offline replay (30/30 passed; 15 evaluable pairs).
- [ ] After Haiku is complete, apply the workflow to Codex Luna.
  Hold: Haiku is complete; the separate Codex Luna native adapter, exact API model contract and credential preflight remain to be delivered. No automatic model fallback.

## Structured reply correction
- [x] Reproduce fenced-text rejection and absent-structured-output fallback with failing regressions.
- [x] Require native JSON schema for new execution; preserve legacy artifacts as replay-only.
- [x] Bound schema retries separately and prove schema/no-fallback mutations.
- [x] Complete independent correction review and clean-tree verification before retry (4d473a4; exit 0).
