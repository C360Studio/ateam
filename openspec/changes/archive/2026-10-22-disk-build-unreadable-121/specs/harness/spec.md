## MODIFIED Requirements

### Requirement: Two-tool MCP context budget

`ateam mcp` SHALL expose exactly two tools (`ateam_state` and `ateam_task`) whose serialized JSON schema is strictly under 300 tokens (~1,200 bytes), alongside zero-turn-tax MCP Prompts and Resources.

#### Scenario: MCP tools/list token check

- **WHEN** a client requests `tools/list` from `ateam mcp`
- **THEN** the response contains `ateam_state` and `ateam_task` and its estimated token count (`bytes / 4`) is `< 300`

#### Scenario: MCP server build provenance and stale binary warning in pickup

- **WHEN** `ateam mcp` handles a `pickup` tool call or resource read
- **THEN** it captures `state.BuildRevision()` at server startup, reports `build=<disk-sha> mcp=<server-sha>` in the `[ateam]` pickup header line (`build=absent` when there is no `./bin/ateam`, `build=unreadable` when its `--version` fails or names no build, in which case staleness is judged against checkout `HEAD`; `ateam doctor` reports the same label as `disk_build` and warns on `unreadable`), and appends `WARNING: MCP server binary is stale; restart the client to pick up <sha>` when the running server's build commit differs from the `./bin/ateam` binary on disk or from checkout `HEAD`
