# Change: Update documentation for Antigravity (agy), 500k context defaults, and recent merged PRs

Issue: https://github.com/C360Studio/ateam/issues/36
Owner: Gemini (`gemini/docs-update`)
implemented-by: gemini

## Why

Several capabilities have landed or been added since the last documentation update, including native Antigravity (`agy`) hook and workspace MCP plugin support, default ~500k token context/auto-compaction thresholds across client configurations, optional `herdr` integration, test-to-spec attribution, stronger-model review routing, and paired benchmark experiments/calibration.

## What Changes

- Update `README.md` and `.agents/README.md` (`internal/bootstrap/templates.go`) to cover Antigravity (`agy`), ~500k context/auto-compaction defaults, `~/.local/bin/ateam` symlink setup, `ateam state herdr`, `ateam bench run|compare|rescore`, `ateam doctor` binary/server checks, and the expanded repository layout.
- Update `docs/hook-flow.md` to document Antigravity (`--agent agy`) event normalization (`PreInvocation`, `PostInvocation`, `Stop`), `conversationId` fallback, `injectSteps` and `{"decision":"allow"}` outputs, platform limitations (`// shortcut:`), ~500k auto-compaction defaults, stale-handoff omission, session-owned compaction checkpoints, and parent-PID liveness checks.
- Update `docs/mcp.md` to include Antigravity workspace plugin MCP registration (`.agents/plugins/ateam/{plugin.json,mcp_config.json}`) and the default 500k context settings in `.codex/config.toml` and `.gemini/settings.json`.
- Update `docs/adoption.md` to reflect Antigravity support, ~500k context defaults in `ateam init`, binary/server checks in `ateam doctor`, and paired benchmark experiment/calibration capabilities in `ateam bench`.

## Impact

- Scope: `README.md`, `.agents/README.md`, `internal/bootstrap/templates.go`, `docs/hook-flow.md`, `docs/mcp.md`, `docs/adoption.md`, and `openspec/changes/docs-latest-work/`.
- Non-goals: Modifying runtime behavior.
