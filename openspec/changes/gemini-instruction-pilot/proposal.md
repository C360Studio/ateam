# Gemini instruction pilot

## Why

The owner reports a paid Gemini tier and authorizes continuing with a bounded availability check, A/A calibration and then an instruction-bundle A/B. Prior Gemini failures remain retained; a changed billing tier alone does not establish capacity or calibration.

## What Changes

Add frozen minimal/ATeam-subset instruction bundles, a predeclared experiment protocol and sanitized results using the existing runner. Track issue #30 under roadmap #16. Native skill work #28 / PR #29 remains separate.

## Scope and stop boundary

Run canary, inspect evidence, then A/A, inspect calibration, then A/B if gates pass, with a combined $2 target and explicit token/time limits. Stop within a phase on native execution/accounting failure or exhausted budget; grading infrastructure errors block later phases after inspection. Preserve evidence and do not retry automatically. Deliver reviewed configurations/protocol/results in a PR, without merging. No runner, native skill/hook, routing or UI changes. No raw traces or secrets are published.

## Owner-authorized v2 amendment

After the v1 canary failure, the owner raised the combined target to $5. `gemini-output-budget` adds the minimal explicit shared output control in this PR. The v2 protocol preserves the treatment text and sample size, uses fresh evidence, and advances only after new gates. V1 files and results stay frozen; the original no-runtime-change scope applied to v1.

## Owner-authorized v3 reconciliation and pilot

Reconcile PRs #43 and #32 with main and run a fresh canary → A/A → A/B within the existing combined $5 target. V3 pins stable CLI 0.64.0, preserves the output allowance/model/reasoning/sample and adds the same self-contained-task sentence to both arms. The candidate-only six-point treatment remains unchanged. See the frozen v3 protocol for accounting, advancement and no-retry rules; prior cohorts remain separate. Protected merges require the repository’s actual required check and must not be bypassed.
