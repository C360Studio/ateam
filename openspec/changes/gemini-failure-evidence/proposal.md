# Gemini fatal failure evidence

## Why

Issue #37 follows the real stderr-only CLI exit53 in the instruction pilot (#30 / PR #32). Raw evidence and stopping were correct, but the diagnostic exposed only an exit code. Complete missing native usage is separately blocked in #38; local telemetry cannot certify a complete fatal tail.

## What changes

Retain optional, strictly recognized Gemini turn-limit error evidence from bounded redacted stderr, with a clear local-limit diagnostic. Add exact execution/stop/replay and refusal regressions. Preserve historical replay and existing CLI settings, turn limits, source grading and accounting rules.

## Scope and stop boundary

Deliver a reviewed implementation and offline proof in a draft PR, without merging or paid calls. Do not enable telemetry, alter tool/model policy, increase turns, repair output or fabricate missing usage. This resolves #37, not the #38 native-accounting blocker or the held A/A/A/B pilot.
