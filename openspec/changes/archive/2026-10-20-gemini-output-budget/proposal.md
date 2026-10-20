# Explicit Gemini output allowance

## Why

The instruction pilot canary retained incomplete JSON near its fixed 8192 output-token limit. The owner authorized $5 pilot headroom. Make this shared execution control explicit so a revised, predeclared pilot can allow thinking and final output without changing model or reasoning between arms.

## What Changes

Add optional bounded `gemini.max_output_tokens` with a legacy-compatible default. Retain it in both native generation scopes, resolution and replay. Revise the pilot as a separate v2 cohort; keep the original v1 evidence and treatments unchanged.

## Scope and stop boundary

Extends issue #30 / draft PR #32 under roadmap #16. Implement, verify and independently review the control, then run the bounded v2 canary, A/A and A/B only as their gates permit, within the combined $5 target. Deliver a reviewed report or exact blocking evidence; do not merge. No native skill/hook, model/reasoning selection, routing or UI expansion. No source or JSON-output repair to turn failed responses into passes.
