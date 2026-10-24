## 1. Implementation

- [ ] 1.1 Attribute live `ateam mcp` processes to this root (registry entry, or working directory equal to the root or a linked worktree); foreign processes appear in doctor's `running_mcp_processes` as `repo: other` and produce no doctor warning or hook line; an unreadable working directory keeps today's behaviour (#151).
- [ ] 1.2 Gate the hook's `MCP server PID n (build x) is older than HEAD` line on `bootstrap.IsAteamCheckout(root)` (#151 comment 6388138323).
- [ ] 1.3 Tests: a `state` table test with a fake process lister (a PID registered under another root and a PID with a foreign working directory produce no warning; a PID with this root's working directory and no registry entry still warns), and a hook test that a foreign checkout prints no `older than HEAD` line. `task mutate:check`: dropping the working-directory comparison and dropping the `IsAteamCheckout` gate must each be `detection`.
- [x] 1.4 Spec delta: ADDED requirement `MCP process attribution` in `specs/harness/spec.md`.
