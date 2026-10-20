# Change: Default 500k context compaction limits in scaffolded client configs

Issue: https://github.com/C360Studio/ateam/issues/35
Owner: Gemini (`gemini/context-window-limits`)
implemented-by: gemini

## Why

On 1M-token context models, allowing active conversation history to grow past 400k-600k tokens degrades attention over long agent trajectories. Newly initialized repositories (`ateam init`) currently scaffold `.claude/settings.json`, `.codex/config.toml`, and `.gemini/settings.json` without default context or compaction bounds, and `ateam doctor` (`CheckParity`) does not validate malformed or out-of-range context limit settings.

## What Changes

- Scaffold default 500k context and compaction thresholds in `internal/bootstrap/templates.go` and the repository's own `.claude/settings.json`, `.codex/config.toml`, and `.gemini/settings.json`:
  - `.claude/settings.json`: `"autoCompactWindow": 500000`
  - `.codex/config.toml`: root-level `model_context_window = 500000` and `model_auto_compact_token_limit = 450000`
  - `.gemini/settings.json`: `"model": { "chatCompression": { "contextPercentageThreshold": 0.5 } }`
- Validate configured context/compaction fields in `CheckParity` (`checkContextConfigs` in `internal/bootstrap/mcp.go`) while continuing to preserve existing client files unchanged and permitting custom valid values or omitted fields.

## Impact

- Affected code: `internal/bootstrap/templates.go`, `internal/bootstrap/mcp.go`, `internal/bootstrap/mcp_test.go`, `.claude/settings.json`, `.codex/config.toml`, `.gemini/settings.json`, and `openspec/specs/harness/spec.md`.
- Non-goals: Antigravity (`agy`) has no user-configurable compaction threshold in `settings.json`, and `InitRepo` continues to preserve pre-existing client files byte-for-byte (`TestInitPreservesClientSettings`).
