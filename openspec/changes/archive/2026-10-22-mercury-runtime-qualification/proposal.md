# Minimal Mercury benchmark integration

## Why

The owner redirected PR #77 from standalone Python qualification infrastructure to the existing Go benchmark runner. Prior offline work established bridge forwarding and raw usage requirements; the next deliverable must run the existing single-file benchmark tasks.

## What Changes

Add an optional Mercury configuration to the existing Claude runtime, a Go-owned per-attempt LiteLLM bridge using existing process cleanup, an in-process Go loopback capture proxy, and Go accounting/replay tests. Reuse `--dry-run`, budgets, scheduling and grading. See [scope](integration-plan.md).

## Impact

Addresses #58 and #16; #28/#59 remain separate. No new runtime, Python supervisor, global configuration or paid run. Preserve archived research and frozen Gemini evidence. Stop at a verified review-ready PR and a proposed $1 canary/A/A requiring separate authorization.
