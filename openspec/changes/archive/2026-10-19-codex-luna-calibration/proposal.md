# Codex Luna API calibration

## Why
Haiku now has a complete A/A pilot. The user requests Codex Luna next, using the same API-only, independently reviewed workflow while keeping model routing outside benchmark MVP scope.

## What Changes
- Add a native Codex adapter with fixed gpt-6-luna and shared explicit effort across both arms.
- Reuse independent workspaces, bounded processes, strict structured replies, offline Go graders and immutable comparison/replay.
- Preserve native accounting and distinguish configured selection from observed model identity. Unknown telemetry must not become a fabricated value or successful fixed-model evidence.
- Add A/A and A/B examples, verification and a separate bounded live calibration report when the API credential and telemetry contract are ready.

## Impact
The existing issue #10 / draft PR #13 remains the claim. Scope is internal/bench, native CLI dispatch, examples and benchmark/OpenSpec documentation. No UI, routing, arbitrary skill imports, automatic model fallback or model-ranking claims. No credentials are copied into this worktree or published.
