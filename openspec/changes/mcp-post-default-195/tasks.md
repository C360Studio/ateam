## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#195). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 The `post` property of the `ateam_state` schema describes the omitted/`true` PR-comment side effect and `false` as local-only, within the 300-token budget (#195). Evidence: `Tools()` in `internal/mcpserver/server.go`; `TestMCPTokenBudget` reports 212 tokens (190 before).
- [x] 2.2 The canonical handoff skill and its `templates.go` copy tell agents to pass `post:false` for local-only and `post:true` when posting is authorized; `ateam doctor` parity stays green (#195). Evidence: `.agents/skills/handoff/SKILL.md` and the `handoff` entry in `internal/bootstrap/templates.go` gain the same sentence, which also names the CLI's `--no-post`; `ateam doctor --check` exits 0.
- [x] 2.3 A test checks the `tools/list` `post` description, proven able to fail by `task mutate:check` (#195). Evidence: `TestMCPToolsListDescribesPostDefault` in `internal/mcpserver/server_test.go`. `task mutate:check` with the description removed reported `detection`; the SHA-256 of `server.go` was the same before and after.
