# Proposal: doctor-relative-path-warning-378

Approved-by: Coby Leuschke (2026-11-16) sha256:968417bb2154

## Track

`quick` (touches `harness` capability only; adds a non-blocking warning in `ateam doctor` for relative hook/MCP command paths per `#378`).

## Why

`ateam doctor` accepts any hook command whose program is `ateam` or ends in `/ateam`, and any `ateam` MCP registration whose command base name is `ateam` or `ateam.exe`. A relative path with a slash (for example `scripts/ateam`) passes both checks, but resolves against the hook's or MCP server's working directory rather than `PATH`, exiting 127 when a session starts below the checkout root (for example in `ui/`).

## What Changes

1. Add `bootstrap.CheckRelativeCommandPaths(root)` (called by `runDoctor` in `cmd/ateam/main.go` and appended to `warnings`) to warn—never fail `--check`—when:
   - A hook command in the active hook file for each client (`.claude/settings.json` or its `ATEAM_HOOK_SCOPE=user` source, `.codex/hooks.json` or its user source, `.gemini/settings.json` or its user source, `.agents/hooks.json`) names its `ateam` program (`tokens[0]` of `hookWords(cmd, "")`) as a relative path containing `/` that does not start with `$` or `~/`.
   - An `ateam` MCP registration in `.mcp.json`, `.codex/config.toml`, `.gemini/settings.json`, or `.agents/plugins/ateam/mcp_config.json` names `command` as a relative path containing `/` that does not start with `$` or `~/`, unless the MCP server entry sets a non-empty `cwd`.
2. Each warning names the file, the relative command, the start-at-checkout-root constraint, and `https://github.com/C360Studio/ateam#install-a-release-adopters`.
3. Update `openspec/specs/harness/spec.md`, `README.md`, and `docs/mcp.md`.

## Scope & Non-Goals

- **In-bounds:** `internal/bootstrap/mcp.go`, `cmd/ateam/main.go`, `internal/bootstrap/mcp_test.go`, `openspec/specs/harness/spec.md`, `README.md`, `docs/mcp.md`.
- **Non-goals:** Changing `CheckParity` violations or failing `ateam doctor --check` on relative paths; warning on bare `ateam`, absolute paths, `~/` paths, `$VAR` paths (`"$CLAUDE_PROJECT_DIR"/…`, `"$GEMINI_PROJECT_DIR"/…`), or MCP entries with an explicit `cwd`.

## Definition of Done

- `task verify`, `ateam doctor --check`, `ateam bench --selftest`, and `task mutate:check` pass.
- OpenSpec change archived before marking the PR ready.
