# Claude Haiku benchmark adapter

## Why
Gemini A/A is blocked by repeated provider HTTP 503 failures. The user now requests Claude Haiku through API authentication first, then Codex Luna after Haiku is complete. Continue the existing issue #10 / draft PR #13 claim; this explicit scope extension supersedes the original single-provider non-goal.

## What Changes
- Add a native Claude Code adapter with a fixed Haiku model and shared effort across both arms.
- Reuse isolated attempts, offline graders, budget/time stops, comparison and immutable replay.
- Retain provider-native session, terminal status and per-model usage without treating missing counters as zero.
- Prepare a bounded A/A example and record verification; run only when API credentials and policy preflight are available.

## Impact
Touches internal/bench execution, configuration, parsing and replay, cmd/ateam native self-test dispatch, examples and benchmark documentation. Gemini and generic adapters remain supported. No routing, UI, arbitrary skill imports, model-ranking claims or automatic fallback. Luna follows successful Haiku delivery rather than changing models mid-pilot.
