# Change: Native Antigravity (.agents/hooks.json and injectSteps) support

implemented-by: gemini

Issue: https://github.com/C360Studio/ateam/issues/33
Owner: Gemini (`gemini/agy-hooks`)

## Why

`ateam` scaffolds and verifies lifecycle hooks for Claude Code (`.claude/settings.json`), Codex CLI (`.codex/hooks.json`), and Gemini CLI (`.gemini/settings.json`), but Antigravity (`agy`) discovers workspace hooks from `.agents/hooks.json` (`SessionStart`, `PreInvocation`, `Stop`), passes camelCase `conversationId` in stdin JSON, and expects `injectSteps` (`[{"systemMessage":{"systemMessage":"..."}}]`) for context injection and `{"decision":"allow"}` on `Stop`.

## What Changes

- Update `internal/hook/hook.go` and `cmd/ateam/main.go` so `ateam hook --agent agy` (or `antigravity`) reads `conversationId` as a fallback for `session_id`, normalizes `PreInvocation` to `before-prompt`, emits `injectSteps` when context is injected, emits `{}` on no-op `PreInvocation`, and emits `{"decision":"allow"}` on `turn-end` (`Stop`).
- Scaffold `.agents/hooks.json` in `ateam init` (`internal/bootstrap/templates.go`) and verify it in `ateam doctor` (`internal/bootstrap/bootstrap.go`).
- Document the Antigravity hook mapping in `.agents/README.md`, `AGENTS.md`, and `.agents/rules-index.md`, and record the platform limitation (`PreCompact` and prompt-bearing `PreInvocation` are not exposed by Antigravity `hooks.json`) via `// shortcut:`.

## Impact

- Scope: `cmd/ateam/main.go`, `internal/hook/`, `internal/bootstrap/`, `internal/state/state.go`, `.agents/hooks.json`, `.agents/README.md`, `AGENTS.md`, `.agents/rules-index.md`, and `openspec/`.
- Non-goals: Workspace MCP plugin bundling under `.agents/plugins/`.
