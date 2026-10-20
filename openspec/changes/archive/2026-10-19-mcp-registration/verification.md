# MCP registration verification

## Regression and scope

Before implementation, `go test ./internal/bootstrap -run 'TestMCPRegistrationFiles|TestInitPreservesClientSettings' -count=1` failed: Codex's project config file was absent, and force initialization overwrote existing client settings. The corrected implementation creates `.codex/config.toml` and `.mcp.json`, leaves Gemini MCP in its supported settings file, and preserves existing client files even with force.

No hook handler, MCP protocol/action, benchmark, or UI behavior is changed. The init JSON response now explains preserved configuration and points to the migration guide; doctor explicitly reports that no connection was checked. Static MCP validation uses JSON/TOML parsing instead of string occurrence checks.

## Test design

- **Decision:** deterministic examples and an explicit operation history; no new fuzz target. TOML grammar is delegated to a maintained parser rather than implemented here.
- **Rule:** registrations must be in client-owned locations with the expected named stdio command; every existing client file must remain byte-identical across initialization, including force.
- **Inputs:** quoted TOML table, multiline args/comments, disabled/wrong/missing entries, misleading unrelated tables/strings, malformed JSON/TOML, wrong argument types/count, remote transport, existing unrelated settings/custom command, and a dangling config symlink.
- **Run:** registration/parity tests and an init history `[normal, normal, force, normal]` compare against literal original file bytes as an independent reference model. Each case creates/changes the targeted config and calls the real public init/parity functions. Both positive and negative assertions execute; tests do not skip on empty data. Exact final revision/gate results are recorded in the PR.
- **Mutation:** Go overlay mutants disable client-file preservation and suppress registration violations; both were detected by the expected assertions, with passing baseline/after checks. One run each, seed 1; see PR for the final source checksum and commands.
- **Not covered:** arbitrary client configuration semantics, trust/global overrides, model-driven tool selection, or all future CLI versions. Local parity is explicitly narrower than integration certification.

## Integration evidence

Used a disposable Git project under the worktree's ignored `.enjoy-logs/`, initialized by the changed binary, with the binary supplied through a subprocess-only PATH. No global agent configuration, trust settings, credentials, or model calls were used/changed.

- **Codex 0.160.1:** the project layer was correctly located by `config/read`, but disabled because the disposable project is not trusted. `codex mcp get ateam` consequently did not discover it. This is not recorded as successful project activation. A bounded foreground app-server probe then supplied the reviewed generated command/args as explicit per-process config; `mcpServerStatus/list` connected to ATeam 0.1.0 and returned both tools and all four resources with no tools error. This proves the actual Codex MCP client can connect, not that project trust was granted. The process was stopped after the probe.
- **Claude Code 2.1.292:** `claude mcp get ateam` discovered the generated project `.mcp.json` registration and reported pending approval. Its user config directory was isolated for the test. No approval was granted; connection remains unverified.
- **Gemini CLI 0.43.0:** `gemini mcp list` discovered the generated stdio registration but reported it disabled because the disposable folder was untrusted. Connection remains unverified.
- **Direct protocol:** 15 operations passed: initialize, tools/prompts/resources listing, prompt retrieval, resource reading, task listing/harmless target execution, and all seven state actions. The initialized notification produced no response. Handoff used `post:false` and returned the expected skipped-post status. The task output matched a literal marker; inventory counts were two tools, four prompts, four resources.

User-visible activation still requires the installed binary on the agent's PATH, the intended repository working directory, and normal client trust/approval. This change does not grant those permissions automatically or reconnect the current conversation.
