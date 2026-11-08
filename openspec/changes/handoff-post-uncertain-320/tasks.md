## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#320). Hold: OPEN-Q proposal awaits owner approval

## 2. Fix

- [ ] 2.1 `PostPRHandoffComment` (`internal/state/forge.go:874`) wraps a send-step timeout in `ErrPostUncertain` on GitHub (`:885-895`) and Gitea (`:951`) (#320, design D1). Acceptance: a test per forge with a send that outlives its deadline gets `errors.Is(err, ErrPostUncertain)`; a 4xx and a missing PR do not.
- [ ] 2.2 Before posting, it reads the PR's comments under a 2 s budget and returns `Posted, Existing` with the newest handoff comment's URL when that body equals the new one (#320, design D2, D3). Acceptance: a test per forge covers equal newest (no POST), different newest (POST), older equal then different (POST), and a failing read (POST); `task mutate:check` reports `detection` with the comparison removed.
- [ ] 2.3 The CLI (`cmd/ateam/main.go:390`) prints `forge PR comment uncertain: …; rerun ateam state handoff to post it once` and MCP (`internal/mcpserver/server.go:342`) prints `PR comment: uncertain (…)`, and `posted (already on PR)` for an existing comment; `docs/hook-flow.md:82` names both (#320). Acceptance: CLI and MCP tests assert the text; `task mutate:check` reports `detection` with uncertain printed as skipped.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#320).
