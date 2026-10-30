## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1 and 2 (#265).
      Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `ownsWriterLease` beside `isCallerActiveWriterLease` (`internal/state/state.go:672`) per D1. Acceptance: the table test from the design's test plan; `task mutate:check` replacing the call with `true` reports `detection`.
- [ ] 2.2 `WriteWriterLease` under `WithLock` (`state.go:1374`) per D3; the `beforeFlock` seam in `WithLock` (`state.go:1588`). Acceptance: the race test passes, and it fails with the `WithLock` wrap removed.
- [ ] 2.3 The D2 critical section in `internal/state`; delete `ClearWriterLease`. Acceptance: own-lease release, a refused handoff, and a failed-release test.
- [ ] 2.4 CLI (`cmd/ateam/main.go:346-361`) and MCP (`internal/mcpserver/server.go:322-332`) call it, with D4 reporting. Acceptance: the #265 reproduction tests for a different agent and for the same agent with a different session, through both paths.
- [ ] 2.5 `task verify` green on a clean tree, `ateam doctor` parity green with `hook_latency_us` recorded, `ateam bench --selftest` green.
- [ ] 2.6 Archive the change before marking the PR ready (#265).
