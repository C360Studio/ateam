# MCP setup and use

`ateam mcp` is a local **stdio** server: the agent client launches the executable and exchanges newline-delimited JSON-RPC over stdin/stdout. It is separate from lifecycle hooks and from the HTTP web board (`ateam serve`). There is no MCP HTTP endpoint or standalone daemon to start first.

It offers two tools, four prompts, and four resources. The two-tool design keeps discovery small by grouping related state operations under an `action` argument. The reported schema budget is a bytes/4 estimate for tool definitions (currently 759 bytes, 190 tokens, against a limit of 300), not a measured tokenizer count or the cost of returned content.

## Install and register

Build ATeam using the [README](../README.md#prerequisites-and-installation), then put the binary on the **agent process's PATH**. A terminal-only PATH change may not reach an already-running desktop app. Alternatively, use an absolute executable path in the registration. Restart/reconnect the agent after changing its configuration.

Run `ateam init` in the repository to create missing project files:

| Client | MCP registration | Hook configuration (separate concern) |
| --- | --- | --- |
| Codex | `.codex/config.toml`, `[mcp_servers.ateam]` | `.codex/hooks.json` |
| Claude Code | `.mcp.json`, `mcpServers.ateam` | `.claude/settings.json` |
| Gemini CLI | `.gemini/settings.json`, `mcpServers.ateam` | `.gemini/settings.json` |
| Antigravity (`agy`) | `.agents/plugins/ateam/{plugin.json,mcp_config.json}`, `mcpServers.ateam` | `.agents/hooks.json` |

Codex project configuration requires a trusted project. Claude Code project servers require the client's normal approval/trust process. Gemini also suppresses servers in untrusted folders. Antigravity loads workspace plugins from `.agents/plugins/<name>/`. Initialization does not grant trust, approve tools, install the binary, or change global configuration.

Codex registration (`.codex/config.toml`, alongside the default 500k context limits):

```toml
model_context_window = 500000
model_auto_compact_token_limit = 450000

[mcp_servers.ateam]
command = "ateam"
args = ["mcp"]
```

Claude Code `.mcp.json`:

```json
{
  "mcpServers": {
    "ateam": { "type": "stdio", "command": "ateam", "args": ["mcp"] }
  }
}
```

For Gemini, merge `"ateam": {"command":"ateam","args":["mcp"]}` into the top-level `mcpServers` object in `.gemini/settings.json` (which `ateam init` also populates with `"context": {"fileName": "AGENTS.md"}` and `"model": {"chatCompression": {"contextPercentageThreshold": 0.5}}`).

Antigravity workspace plugin (`.agents/plugins/ateam/plugin.json` and `.agents/plugins/ateam/mcp_config.json`):

```json
// .agents/plugins/ateam/plugin.json
{ "name": "ateam" }

// .agents/plugins/ateam/mcp_config.json
{
  "mcpServers": {
    "ateam": { "command": "ateam", "args": ["mcp"] }
  }
}
```

The server uses its process working directory to locate the repository. Start the client from the intended checkout; where the client supports `cwd`, set it explicitly if needed (Codex/Gemini support it). An absolute executable path does **not** select a repository. Verify the branch/HEAD in a pickup result, especially when using worktrees or a desktop client. Registering globally without considering working directory can target the wrong checkout.

Registration formats are based on [Codex MCP documentation](https://developers.openai.com/codex/mcp/), [Claude Code project scope](https://code.claude.com/docs/en/mcp#project-scope), [Gemini MCP settings](https://geminicli.com/docs/tools/mcp-server/), and Antigravity workspace plugin conventions (`.agents/plugins/`). Check the docs for your installed client version and its policy overrides.

## Migrating existing configurations

Earlier ATeam versions embedded `mcpServers` in `.codex/hooks.json` and `.claude/settings.json`. Those entries are not the project registration locations above.

1. Run `ateam init` without `--force`. Missing `.codex/config.toml`, `.mcp.json`, and `.agents/plugins/ateam/{plugin.json,mcp_config.json}` files are created.
2. If a destination already exists, **merge just the ATeam stanza manually**, preserving other servers, permissions, comments, and custom paths. Do the same for an existing Gemini settings file that lacks ATeam. Existing client configuration is left byte-for-byte unchanged even with `--force`; that option only replaces other generated files such as documents and skill adapters.
3. Remove only the obsolete `mcpServers.ateam` entry from the old Codex/Claude hook/settings files after reviewing their other contents. Keep hook wiring and unrelated settings intact. ATeam does not rewrite existing files to perform this cleanup.
4. Run `ateam doctor --check`, then the client checks below. If doctor reports missing registration in a preserved file, manually merge the examples; rerunning `--force` will not erase the file to resolve it.

Doctor parses JSON and TOML at the project locations and checks an enabled stdio registration named `ateam`, with command `ateam` (or a path ending in `ateam`/`ateam.exe`) and exactly `args = ["mcp"]`. Wrapper commands and differently named executables are outside this local parity convention. It does not resolve PATH, certify the whole client configuration, inspect global overrides, validate credentials/trust, or start a server. Its output explicitly reports `mcp_connection_checked: false`.

## Verify in layers

1. **Configuration:** `ateam doctor --check` validates local structure. `codex mcp get ateam --json`, `claude mcp get ateam`, and `gemini mcp list` help inspect what clients discover. A client's CLI/configuration scope may differ from an active session; inspect that session too.
2. **Connection:** use the client's MCP status UI (for example `/mcp`) after normal trust/approval and restart. A pending-approval or disabled entry is discovery evidence, not a working connection. No ATeam tools in the active session means the integration is not ready in that session.
3. **Operations:** call `ateam_task` with `{"target":"list"}` and `ateam_state` with `{"action":"pickup"}`. Confirm the expected repository and task names. In a disposable project, run a harmless known target and verify its output. These are not pure reads: state refresh writes local cache, and task execution runs the project's commands.

For a direct protocol smoke check from the repository, independent of any agent client:

```sh
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ateam mcp
```

Expect initialization followed by exactly two tool definitions. EOF stops this test server. This checks the binary/protocol path, **not** client discovery, trust, or live agent use. The current server advertises protocol version `2024-11-05`; broader protocol-version compatibility is not certified here.

## Tool reference

`ateam_state` requires `action`. Every action except `read_handoff` first refreshes local repository state and saves `.ateam/state.json`; `pickup` and `board` also attempt a bounded remote forge sync when a remote forge is configured (falling back to cached cards on failure), and `ateam state sync` performs an explicit CLI sync.

| Action | Arguments and result | Additional effects |
| --- | --- | --- |
| `pickup` | `{"action":"pickup"}` returns a human-readable reconciliation summary including `build` and `synced` timestamps, the worktree list with PR status, and the handoffs recorded inside other worktrees. | Refreshes local state and attempts a bounded remote sync when stale. |
| `read_handoff` | `{"action":"read_handoff","slot":"claude@main"}` returns that slot's five fields from this checkout's cache, then linked worktrees' caches (this checkout wins a duplicate), under a `STALE — <reasons>` banner when stale. A missing slot is a tool error listing the slots that exist. | None: no refresh, cache write, writer-lease change, persisted identity change, or forge call. Staleness uses live heads and the session from env or the lease, not `session_id`. |
| `debt` | Returns scanned shortcuts as JSON text. | Limited to [supported extensions](adoption.md#feature-and-language-boundaries). |
| `mode` | Without `mode`, reads current mode; with `"mode":"lite"`, `"full"`, `"ultra"`, or `"off"`, changes it and returns mode/ladder text. | Saves mode when supplied; does not change provider/model settings. |
| `spec_queue` | Returns in-flight OpenSpec task/caveat summary. | Local refresh only. |
| `spec_check` | Validates OpenSpec and reports tool error on invalid specs. | May invoke installed OpenSpec CLI. |
| `board` | Returns board cards as JSON text. | Attempts a bounded remote forge sync when stale, falling back to cache on failure. |
| `handoff` | Accepts `goal`, `addresses`, `constraints`, `done_when`, `checks_to_run`, and optional boolean `post`. Returns the five-field checkpoint and posting status. | Saves checkpoint, attempts to clear the writer lease, and **defaults to attempting a PR comment** when a forge is configured. Set `post:false` for local-only use. |

Example local-only handoff:

```json
{
  "action": "handoff",
  "goal": "Finish issue #11",
  "addresses": "PR #12",
  "constraints": "No global config changes",
  "done_when": "Registration and documentation verified",
  "checks_to_run": "task verify",
  "post": false
}
```

Supply meaningful handoff fields: the current MCP schema only requires `action`, unlike the CLI's required `--goal` check. A skipped/failed PR comment is described in the returned text; a locally saved checkpoint can still succeed.

`ateam_task` accepts optional `target` and `args` (array of strings). Empty target or `"list"` lists targets from `Taskfile.yml`. Otherwise it validates the target name and invokes `task <target>`, appending `--` and supplied arguments. The runner has a five-minute timeout. A listed target can still be destructive or contact external services: authorize its actual commands before running it. There is no verification or sandbox bypass built into MCP.

```json
{"target":"test:unit"}
```

Tool results contain text content and `isError`. JSON results such as board cards are encoded inside that text. Clients should inspect `isError`; a JSON-RPC response alone does not mean the task succeeded.

## Prompts and resources

Prompts `pickup`, `handoff`, `preflight`, and `ponytail` return the corresponding `.agents/skills/<name>/SKILL.md` as user-role prompt text. Fetching a prompt does not execute its workflow or checks. Prompt availability depends on those files in the selected repository.

| Resource URI | Contents |
| --- | --- |
| `ateam://state/pickup` | Pickup summary |
| `ateam://state/board` | Cached board cards as JSON text |
| `ateam://state/shortcuts` | Shortcut scan as JSON text |
| `ateam://state/openspec` | OpenSpec queue summary |

Resource reads also refresh and save local state. Client support for prompts/resources varies independently of tool support; the two tools do not automatically inject all these contents into every turn.

## Troubleshooting

- **Stale server after a rebuild:** an `ateam mcp` process keeps the binary it started from. The `session-start` hook (in the ateam checkout) prints `MCP server PID <n> (build <commit>) is older than HEAD; restart the client`, and `ateam doctor` warns `running ateam mcp process (PID <n>, build <commit>) is older than <commit>; restart the MCP client` for processes serving this repository (compared with `HEAD` in the ateam checkout, otherwise with the on-disk `bin/ateam`). Processes serving other repositories are listed in `running_mcp_processes` with `"repo": "other"` and not warned about. Restart or reconnect the client; `ateam version` shows the build you would get.
- **Executable not found:** check the environment inherited by the agent, or use an absolute command path. Reconnect after changing it.
- **Registration absent:** check the client-specific file, correct nesting, project trust, and global/managed overrides. An entry in a hooks file is insufficient for Codex/Claude MCP registration.
- **Pending/disabled:** complete the client's normal trust/approval process; doctor does not grant it.
- **Wrong repository or missing Taskfile:** inspect launch working directory and the pickup result. ATeam initialization does not create your project's verification tasks.
- **Protocol succeeds but tools are unavailable:** the binary works; inspect the agent's connection status and restart. Do not equate the direct smoke check with an active session connection.

Implementation: [bootstrap](../internal/bootstrap/mcp.go), [server](../internal/mcpserver/server.go), [task execution](../internal/taskrunner/taskrunner.go). Actual tested client versions and limitations for this change are recorded in the PR verification evidence.
