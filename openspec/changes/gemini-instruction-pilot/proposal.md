# Gemini instruction pilot

## Why

The owner reports a paid Gemini tier and authorizes continuing with a bounded availability check, A/A calibration and then an instruction-bundle A/B. Prior Gemini failures remain retained; a changed billing tier alone does not establish capacity or calibration.

## What Changes

Add frozen minimal/ATeam-subset instruction bundles, a predeclared experiment protocol and sanitized results using the existing runner. Track issue #30 under roadmap #16. Native skill work #28 / PR #29 remains separate.

## Scope and stop boundary

Run canary, inspect evidence, then A/A, inspect calibration, then A/B if gates pass, with a combined $2 target and explicit token/time limits. Stop within a phase on native execution/accounting failure or exhausted budget; grading infrastructure errors block later phases after inspection. Preserve evidence and do not retry automatically. Deliver reviewed configurations/protocol/results in a PR, without merging. No runner, native skill/hook, routing or UI changes. No raw traces or secrets are published.
