# Tasks

## Implementation
- [x] Review plan independently and inspect installed Claude CLI / official accounting contract.
- [x] Implement API-only isolated Claude execution and native evidence parsing.
- [x] Integrate native configuration, stop rules, replay and CLI self-test dispatch.
- [x] Add A/A and A/B examples and explain the fixed-model workflow.

## Verification
- [x] Prove focused parser/execution/accounting regressions and mutations; record generated-check decision.
- [ ] Complete independent change review and clean-tree task verify.
- [ ] Publish implementation and evidence through draft PR #13.

## Live calibration
- [ ] Run and report Haiku A/A with immutable artifacts and offline replay.
  Hold: ANTHROPIC_API_KEY is absent; require isolated remote-policy preflight before live inference. Provider accounting and cost remain unverified until live evidence is available.
- [ ] After Haiku is complete, apply the workflow to Codex Luna.
  Hold: Sequenced after Haiku completion per user instruction; no automatic model fallback.
