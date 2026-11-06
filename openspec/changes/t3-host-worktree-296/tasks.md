## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#296).
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Name T3 Code in the Claim exemption in `AGENTS.md`, `.agents/protocol.md`, and `protocolClaimStep` (`internal/bootstrap/templates.go`); the drift test covers the sentence (#296).
- [ ] 2.2 Pickup names the owner of a worktree on an unprefixed branch from its handoff slot, else its writer lease, else `owner unknown` (#296). Acceptance: table test for all three; `task mutate:check` swapping the order reports `detection`.
- [ ] 2.3 Prune keeps an unprefixed branch as `no agent prefix (host-created branch)` (#296).
- [ ] 2.4 `docs/adoption.md` T3 Code paragraph, including the open question on prefix configurability (#296).
- [ ] 2.5 Spec scenarios on the Requirement "Prune finished worktrees the caller owns" (#296).
- [ ] 2.6 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green.
- [ ] 2.7 Archive the change before marking the PR ready (#296).
