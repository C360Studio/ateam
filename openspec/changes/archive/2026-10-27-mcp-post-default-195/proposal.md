# Change Proposal: `mcp-post-default-195`

Approved-by: Coby Leuschke (2026-10-27)

## Why

The MCP `handoff` action posts a PR comment when `post` is omitted (`shouldPost := args.Post == nil || *args.Post` in `internal/mcpserver/server.go`). The tool listing an agent reads before the call does not say so: the `ateam_state` description says only "Reconcile pickup/handoff", and the `post` boolean has no description. The default is explained in `docs/mcp.md` and the protocol, which the agent does not see at the call (#195).

## What Changes

- The `post` property in the `ateam_state` input schema gets a short description: omitted or `true` attempts a PR comment on the branch's open PR; `false` saves locally only.
- The canonical handoff skill (`.agents/skills/handoff/SKILL.md` and its template in `internal/bootstrap/templates.go`) tells the agent to pass `post:false` for a local-only checkpoint and `post:true` when posting is authorized, instead of relying on the default.
- The harness spec's "Accurate MCP integration documentation" requirement gains one scenario for the tool listing.

## Track

`quick`. One existing requirement gains a scenario; no requirement is added. It changes schema text and guidance only: no runtime behavior, boot path, durability, gate file, or new exported surface.

## Non-goals

- Changing the default. Omitted `post` keeps posting; a new default needs its own compatibility decision (#195).
- New tools, actions, or an approval gate for posting.
- Posting a real PR comment in a test.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green.
- A test reads the `tools/list` response and checks the `post` property describes the omitted/`true` PR-comment side effect and `false` as local-only; `task mutate:check` with the description removed reports `detection`.
- The existing `post:false` handoff test still passes, and `TestMCPTokenBudget` stays under 300 tokens (190 today).
