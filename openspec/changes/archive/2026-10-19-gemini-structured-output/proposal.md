# Gemini native output and review corrections

## Why

The latest Gemini canary returned a response after a provider retry, but prose-only instructions produced fenced JSON. Strict parsing correctly rejected it. PR #13 review also found that operational failures were labeled model-unverified and reused sessions were graded before refusal.

## What Changes

- Require native JSON schema output for new Gemini runs, preserving legacy configuration and artifact replay.
- Prefer operational stop reasons for timeout/cancellation/adapter errors; retain existing recorded histories and intentional identity-refusal reasons.
- Refuse reused sessions before executing candidate grading.
- Correct stale adapter documentation and record the new canary separately.

## Scope and delivery

Scope: benchmark configuration/settings, execution/stop validation, relevant fixtures/tests, examples and evidence. No UI, model routing, generic provider-interface refactor or relaxed source parsing. Deliver through existing PR #13 with independent review, regression/mutation proof and full clean-tree verification. Retry a bounded API canary only after verification; proceed to the fixed-model A/A schedule only with complete native accounting. The $2 target remains a target with explicit token/time bounds, not a hard monetary guarantee.
