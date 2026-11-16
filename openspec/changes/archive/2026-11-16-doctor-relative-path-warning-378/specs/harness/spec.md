# harness Specification Delta

## MODIFIED Requirements

### Requirement: Client-readable MCP registration

ATeam initialization SHALL place MCP registration in the supported configuration surface for each supported client, preserve unrelated existing settings, and document any required migration or manual setup. Local parity checks SHALL validate the expected registration structure without representing that result as proof of a live client connection, and `ateam doctor` SHALL warn without failing `--check` when a hook or `cwd`-less `ateam` MCP command uses a relative path.

#### Scenario: Codex MCP discovery

- **WHEN** ATeam is initialized for Codex in a trusted disposable project with its executable available
- **THEN** the registration uses Codex's supported MCP configuration surface and client discovery can identify the ATeam server without relying on an entry in a hooks file

#### Scenario: Existing client configuration

- **WHEN** initialization encounters existing client settings and servers
- **THEN** unrelated configuration is preserved and any required manual migration is explained instead of silently overwriting settings

#### Scenario: Relative hook or MCP command path warning

- **WHEN** `ateam doctor` inspects hook commands (`.claude/settings.json`, `.codex/hooks.json`, `.gemini/settings.json`, `.agents/hooks.json`, or the active `ATEAM_HOOK_SCOPE=user` source) or `ateam` MCP registrations (`.mcp.json`, `.codex/config.toml`, `.gemini/settings.json`, `.agents/plugins/ateam/mcp_config.json`)
- **THEN** it emits a warning naming the file, the command, the start-at-root constraint, and `https://github.com/C360Studio/ateam#install-a-release-adopters` for each `ateam` command that contains `/` and is not an absolute path, a `~/` path, or a host-variable path starting with `$` (such as `"$CLAUDE_PROJECT_DIR"/...` or `"$GEMINI_PROJECT_DIR"/...`), unless an MCP entry sets a non-empty `cwd`, and `--check` still passes on that warning alone
