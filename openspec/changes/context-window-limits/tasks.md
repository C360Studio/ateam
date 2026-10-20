# Tasks: Default 500k context compaction limits in scaffolded client configs

implemented-by: gemini

## 1. Implementation & Verification

- [x] 1.1 Update `claudeSettingsJSON`, `codexMCPConfig`, and `geminiSettingsJSON` in `internal/bootstrap/templates.go` plus repository `.claude/settings.json`, `.codex/config.toml`, and `.gemini/settings.json` with 500k context/compaction defaults.
- [x] 1.2 Add `checkContextConfigs` in `internal/bootstrap/mcp.go` to validate configured client context and compaction bounds during `CheckParity` while allowing omitted or custom valid settings in preserved client configs.
- [x] 1.3 Add `TestClientContextCompactionLimits` in `internal/bootstrap/mcp_test.go` and verify mutation detection (`task mutate:check`), `task test:spec`, `task verify`, and `ateam doctor --check`.
