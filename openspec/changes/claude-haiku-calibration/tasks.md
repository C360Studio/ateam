# Tasks

## Implementation
- [x] Review plan independently and inspect installed Claude CLI / official accounting contract.
- [x] Implement API-only isolated Claude execution and native evidence parsing.
- [x] Integrate native configuration, stop rules, replay and CLI self-test dispatch.
- [x] Add A/A and A/B examples and explain the fixed-model workflow.

## Verification
- [x] Prove focused parser/execution/accounting regressions and mutations; record generated-check decision.
- [x] Complete independent change review and clean-tree task verify.
- [ ] Publish implementation and evidence through draft PR #13.

## Live calibration
- [ ] Run and report Haiku A/A with immutable artifacts and offline replay.
  Hold: API credential and isolated remote-policy preflight are ready. First live pilot exposed fenced text replies; verify the native structured-output correction before retry. Prior 11 completed attempts report 23,075 tokens and $0.288833 CLI estimate with unknown pricing basis; interrupted attempt usage is unknown. Preserve cumulative consumption.
- [ ] After Haiku is complete, apply the workflow to Codex Luna.
  Hold: Sequenced after Haiku completion per user instruction; no automatic model fallback.

## Structured reply correction
- [x] Reproduce fenced-text rejection and absent-structured-output fallback with failing regressions.
- [x] Require native JSON schema for new execution; preserve legacy artifacts as replay-only.
- [x] Bound schema retries separately and prove schema/no-fallback mutations.
- [ ] Complete independent correction review and clean-tree verification before retry.
