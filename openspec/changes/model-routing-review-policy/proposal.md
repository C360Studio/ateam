# Change: Model routing and stronger-model independent review

Addresses #14: https://github.com/C360Studio/ateam/issues/14

implemented-by: codex

## Why

ATeam has lifecycle hooks and bootstrap parity but no model-routing or review-evidence gate. Semengine supplies independent review contracts and provider separation, but its model policy permits equal tiers and its merge script checks provider labels rather than model strength, verdict or commit freshness.

The owner requires a stronger reviewer model as the default; cross-provider review is optional. On 2026-10-19 the owner selected: **Block approval; require an explicit owner exception** when development already uses the highest approved model.

## What Changes

- Define versioned developer/reviewer model policy with explicit superiority comparisons and optional provider separation.
- Require an independent reviewer who contributed no implementation; keep reasoning effort separate from model strength.
- Block eligibility for unknown/unavailable models or unresolved inheritance. Never silently fall back to equal tiers or lower the developer model.
- Scope ceiling exceptions to the owner, reason, reviewed commit, reviewer model and applicable policy revision.
- Port generic review behavior and commit-bound evidence, with a Go validator and Taskfile entrypoint.
- Extend bootstrap/doctor checks and integrate GitHub/Gitea evidence without increasing the two-tool MCP budget or adding network work to hooks.

## Impact

- New capability: `model-review`.
- Intended implementation: new `internal/review` package, CLI and Taskfile entrypoint, role policy/contracts/adapters, bootstrap parity, and narrow forge integration.
- Shared integration files: `cmd/ateam/main.go`, `Taskfile.yml`, `internal/bootstrap/*`, `internal/state/forge.go`, `AGENTS.md`, `.agents/README.md`, `.agents/protocol.md`, `.agents/rules-index.md`.
- No UI, benchmark implementation, automatic agent launcher, universal model leaderboard, or Semengine-specific NATS/graph contract port.

## Delivery boundary

This draft claim contains planning artifacts only. Implementation, independent review and merge remain open tasks. It must not close #14 merely because the plan merges.
