## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#320). Evidence: the owner selected "Approve, keep exit 0 on uncertain" in Enjoy on 2026-11-08; Claude wrote the header at their request.

## 2. Fix

- [x] 2.1 `PostPRHandoffComment` (`internal/state/forge.go:874`) wraps a send-step timeout in `ErrPostUncertain` on GitHub (`:885-895`) and Gitea (`:951`) (#320, design D1). Acceptance: a test per forge with a send that outlives its deadline gets `errors.Is(err, ErrPostUncertain)`; a 4xx and a missing PR do not. Evidence: `TestPostPRHandoffCommentReportsATimedOutSendAsUncertain`; `ateam mutate` with the GitHub `%w` turned to `%v` reported `detection`.
- [x] 2.2 Before posting, it reads the PR's comments under a 2 s budget and returns `Posted, Existing` with the newest handoff comment's URL when that body equals the new one (#320, design D2, D3). Acceptance: a test per forge covers equal newest (no POST), different newest (POST), older equal then different (POST), and a failing read (POST); `task mutate:check` reports `detection` with the comparison removed. Evidence: `TestPostPRHandoffCommentSkipsARepeatOfTheNewestHandoff` (7 cases per forge); `ateam mutate` with the equality forced false reported `detection`.
- [x] 2.3 The CLI (`cmd/ateam/main.go:390`) prints `forge PR comment uncertain: …; rerun ateam state handoff to post it once` and MCP (`internal/mcpserver/server.go:342`) prints `PR comment: uncertain (…)`, and `posted (already on PR)` for an existing comment; `docs/hook-flow.md:82` names both (#320). Acceptance: CLI and MCP tests assert the text; `task mutate:check` reports `detection` with uncertain printed as skipped. Evidence: `TestStateHandoffPostOutcome`, `TestMCPHandoffPostOutcome`; `ateam mutate` on `cmd/ateam/main.go` with the uncertain branch disabled reported `detection`; source SHA-256 unchanged after each run.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#320).
