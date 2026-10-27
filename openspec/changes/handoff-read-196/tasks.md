## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1 and 2 (#196). Evidence: the owner approved in Enjoy on 2026-10-27 (stale slots print under a banner; `read-handoff` / `read_handoff` with `slot` accepted), and Claude wrote the header at their request.

## 2. Preview

- [ ] 2.1 `formatOtherHandoffLine` (`internal/state/state.go:2800`) appends `: <cleanTaskText(goal)>` after the attribution, also for the legacy branch at `state.go:2725-2726`. Acceptance: a test seeds `claude@main`, renders as `codex@main`, and matches the full line; `task mutate:check` with the suffix removed reports `detection`; `TestHookLatencyBudget` passes (#196).

## 3. Read path

- [ ] 3.1 A read function in `internal/state` takes a slot key, reads `Load(root)` (`state.go:1435`) then linked worktree caches as `worktreeHandoffLines` does (`state.go:2849-2868`), and returns the handoff, stale reasons from `handoffStaleReasonsAt` (`state.go:918`), or the existing keys. Acceptance: table test over current-checkout, worktree-only, duplicate (current wins), stale, and missing slots (#196).
- [ ] 3.2 CLI `ateam state read-handoff <agent>@<branch>` (`cmd/ateam/main.go:143-149` dispatch) and `ateam state --help` list it. Acceptance: the regression below passes through the CLI (#196).
- [ ] 3.3 MCP `read_handoff` action with `slot` (`internal/mcpserver/server.go:52-75`, `257-292`) that skips identity resolution and `ReconcilePickup`. Acceptance: `TestMCPTokenBudget` stays under 300 (#196).
- [ ] 3.4 Regression: seed a Claude handoff and a lease, read as Codex through CLI and MCP; assert the cache SHA-256, lease file and resolved Codex identity are unchanged and the forge fake records no call. `task mutate:check` reports `detection` when the read path calls `ReconcilePickup` and when the stale banner is dropped (#196).

## 4. Docs and verify

- [ ] 4.1 `docs/mcp.md` (`:97-103` table) documents `read_handoff`; the spec delta joins `openspec/specs/harness/spec.md` on archive (#196).
- [ ] 4.2 `task verify` green on a clean tree and `ateam doctor` parity green (#196).
