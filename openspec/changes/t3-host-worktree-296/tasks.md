## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#296). Evidence: owner wrote "Approved, quick track" in Enjoy on 2026-11-06; Claude wrote the header from that answer.

## 2. Implementation

- [x] 2.1 Name T3 Code in the Claim exemption in `AGENTS.md`, `.agents/protocol.md`, and `protocolClaimStep` (`internal/bootstrap/templates.go`); the drift test covers the sentence (#296). Evidence: `AGENTS.md`, `.agents/protocol.md`, `protocolClaimStep` and `protocolClaimStepAIP`; `TestInitClaimNamesHostCreatedWorktrees` checks the generated protocol in default and `aip` modes.
- [x] 2.2 Pickup names the owner of a worktree on an unprefixed branch from its handoff slot, else its writer lease, else `owner unknown` (#296). Acceptance: table test for all three; `task mutate:check` swapping the order reports `detection`. Evidence: `worktreeHostOwner` in `internal/state/state.go`; `TestPickupNamesOwnerOfHostCreatedBranch` (handoff, lease, both, neither, plus `main` and a prefixed branch unlabelled); `ateam mutate` swapping lease before handoff reported `detection` at `state_test.go:1732`, `state.go` SHA-256 unchanged.
- [x] 2.3 Prune keeps an unprefixed branch as `no agent prefix (host-created branch)` (#296). Evidence: `internal/state/prune.go:47`; `TestPruneVerdict` row `no agent prefix`.
- [x] 2.4 `docs/adoption.md` T3 Code paragraph, including the open question on prefix configurability (#296). Evidence: `docs/adoption.md` § T3 Code hosts.
- [x] 2.5 Spec scenarios on the Requirement "Prune finished worktrees the caller owns" (#296). Evidence: `specs/harness/spec.md` in this change.
- [ ] 2.6 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green.
- [ ] 2.7 Archive the change before marking the PR ready (#296).
