## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#210). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `SnapshotGit` records the default branch's remote-tracking tip and local ahead count from refs on disk (#210).
- [ ] 2.2 `handoffStaleReasonsAt` marks a default-branch slot handoff stale when its checkpoint is behind the remote tip, keeping a local-ahead checkpoint fresh (#210).
- [ ] 2.3 CLI, MCP, and web handoff warn when written with no resolvable session id (#210).
- [ ] 2.4 `HandoffSlotHeldError` names the holder's writer-lease state when known (#210).
- [ ] 2.5 Tests for each, mutation checks, hook latency via `ateam doctor` (#210).
- [ ] 2.6 Harness spec requirement and scenarios (#210).
