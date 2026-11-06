## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#296). Evidence: owner wrote "Approved, quick track" in Enjoy on 2026-11-06; Claude wrote the header from that answer.

## 2. Implementation

- [x] 2.1 Name T3 Code in the Claim exemption in `AGENTS.md`, `.agents/protocol.md`, and `protocolClaimStep` (`internal/bootstrap/templates.go`); the drift test covers the sentence (#296). Evidence: `AGENTS.md`, `.agents/protocol.md`, `protocolClaimStep` and `protocolClaimStepAIP`; `TestInitClaimNamesHostCreatedWorktrees` checks the generated protocol in default and `aip` modes.
- [x] 2.2 Pickup names the owner of a worktree on an unprefixed branch from its handoff slot, else its writer lease, else `owner unknown` (#296). Acceptance: table test for all three; `task mutate:check` swapping the order reports `detection`. Evidence: `worktreeHostOwner` in `internal/state/state.go`; `TestPickupNamesOwnerOfHostCreatedBranch` (handoff, lease, both, neither, plus `main` and a prefixed branch unlabelled); `ateam mutate` swapping lease before handoff reported `detection` at `state_test.go:1732`, `state.go` SHA-256 unchanged.
- [x] 2.3 Prune keeps an unprefixed branch as `no agent prefix (host-created branch)` (#296). Evidence: `internal/state/prune.go:47`; `TestPruneVerdict` row `no agent prefix`.
- [x] 2.4 `docs/adoption.md` T3 Code paragraph, including the open question on prefix configurability (#296). Evidence: `docs/adoption.md` § T3 Code hosts.
- [x] 2.5 Spec scenarios on the Requirement "Prune finished worktrees the caller owns" (#296). Evidence: `specs/harness/spec.md` in this change.
- [x] 2.6 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green. Evidence: `task verify` exit 0 on a clean tree at `ec3ddff` (`test-spec: all discovered supported tests have valid citations`, `selftest_passed: true`); `ateam doctor --check` `passed: true`, `violations: null`.
- [x] 2.7 Archive the change before marking the PR ready (#296). Evidence: archived to `openspec/changes/archive/2026-11-06-t3-host-worktree-296/`.
- [x] 2.8 Pick the owner from every slot the worktree holds, not the display set that drops slots the caller's cache also holds (Codex review of PR #297, P2). Evidence: `worktreeHostOwner` reads `readWorktreeHandoffSlots(&State{Git: st.Git}, wt)`, as `occupiedBy` does; duplicate-slot case added to `TestPickupNamesOwnerOfHostCreatedBranch` (caller `claude@` 10:00 and `codex@` 11:00, worktree `claude@` 12:00 → `owner claude`); `ateam mutate` restoring the display helper reported `detection` at `state_test.go:1759`, `state.go` SHA-256 unchanged.
