# Change: MCP registration and integration documentation

Issue: https://github.com/C360Studio/ateam/issues/11
Owner: Codex (`codex/mcp-registration`)

## Why

Direct stdio tests work, but the inspected Codex session has no ATeam registration. Generated MCP configuration lives in the hook file rather than the documented Codex MCP configuration surface. Current parity checks can pass without client discovery. Public documentation also lacks a complete MCP guide and advertises an unsupported sync action.

## What Changes

- Verify current client registration formats, then fix bootstrap adapters and local checks accordingly.
- Preserve existing settings and document migration, executable discovery, and repository working-directory requirements.
- Document actual tools/actions, prompts, resources, side effects, and connection checks.
- Separate local parity, protocol functionality, and real-client integration evidence.

## Impact

Expected implementation scope: internal/bootstrap/templates.go, internal/bootstrap/bootstrap.go, bootstrap tests, relevant CLI tests, client configuration files, README.md, .agents/README.md, and a new docs/mcp.md. Inspect internal/mcpserver and task/state handlers as evidence; change runtime MCP behavior only if required by a separately justified finding.

Risks: overwriting user settings, assuming the wrong client configuration format, treating configuration text as connection proof, or connecting a server to the wrong checkout. Use disposable settings/projects for tests. No global settings changes at this planning checkpoint.

## Verification and Stop Boundary

Prove a meaningful failing registration regression before fixing it. Cover settings preservation and idempotence, record mutation/PBT decisions for changed logic, exercise the protocol, and verify actual client discovery/connection where available. Run preflight/parity gates at a recorded revision and label every untested client explicitly.

This initial draft claims the issue and records the plan. Implementation tasks remain open. No merge, runtime changes, or agent-global registration changes at this checkpoint.
