## ADDED Requirements

### Requirement: MCP process attribution

`ateam doctor` and the session-start hook SHALL warn only about `ateam mcp` processes that serve the current repository, and SHALL compare a server's build with checkout `HEAD` only in the ateam checkout.

#### Scenario: Foreign MCP process

- **WHEN** a live `ateam mcp` process is not in this root's registry and its working directory is readable and is neither this root nor a linked worktree of it
- **THEN** doctor lists it in `running_mcp_processes` with `repo: other` and emits no warning for it, and the session-start hook prints no line for it

#### Scenario: Unregistered MCP process in this repository

- **WHEN** a live `ateam mcp` process is not in this root's registry and its working directory is this root, or its working directory cannot be read
- **THEN** doctor and the hook warn about it as before

#### Scenario: Older-than-HEAD comparison outside the ateam checkout

- **WHEN** doctor or the session-start hook runs in a checkout whose root `go.mod` does not declare `module github.com/c360studio/ateam`
- **THEN** the hook prints no `MCP server PID n (build x) is older than HEAD; restart the client` line
- **AND** doctor does not compare a server's build with checkout `HEAD`; it compares with the on-disk `bin/ateam` build, as it does when there is no `HEAD`
