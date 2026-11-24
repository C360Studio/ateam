# Proposal: gemini-neon-hydrogen-405
Approved-by: Coby Leuschke (2026-11-24) sha256:1e9db993a0a3

## Track

`design`: touches a gate file (`.agents/review-policy.json`). No spec Requirement changes. The model-review spec names no model IDs (`openspec/specs/model-review/spec.md:153-186`).

## Why

`agy models` (agy 1.5.14, run 2026-11-24) lists `gemini-4-neon` and `gemini-4-hydrogen` with `low`, `medium`, `high` and `xhigh` variants. `.agents/review-policy.json:4-6` ranks neither, so `ateam review compare` and `merge-check` treat a Neon or Hydrogen run as `unranked` and block it against any ranked run. `effort_bands.gemini` (`.agents/review-policy.json:11`) has no `xhigh`. `gemini-3.1-pro` (`:6`) is no longer offered.

## What Changes

1. `.agents/review-policy.json`: add `gemini:gemini-4-neon` to `workhorse` and `gemini:gemini-4-hydrogen` to `light`, drop `gemini:gemini-3.1-pro`, and add `xhigh` to the Gemini `high` band. These are the owner's rulings on #405, given 2026-11-24.
2. `docs/adr/0002-model-tiers-and-effort-bands.md`: update the § 1 tier table and JSON, the Gemini effort row and band table, and add a Status amendment.
3. `docs/model-review.md`: add a note that agy puts effort in the model ID (`gemini-4-neon-xhigh`), so a PR line splits it: `gemini (gemini-4-neon, xhigh)`. Orchestrator modes stay unranked.
4. `internal/review/compare_test.go:18`: update the committed-policy copy.

## Non-goals

- Parsing effort suffixes out of model IDs.
- Ranking Orchestrator modes.
- Changing the bootstrap code. `ateam init` seeds no tiers, and the issue's mention of `internal/bootstrap/review.go` was wrong.

## Definition of Done

`task verify` passes. `ateam review compare` passes `gemini:gemini-4-neon:xhigh` reviewing `gemini:gemini-4-argon:high`, and fails a `gemini-4-hydrogen` reviewer on Argon work. The PR carries the owner's `gate-change-approved`.
