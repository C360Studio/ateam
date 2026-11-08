## 1. Approval

- [ ] 1.1 The owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, ruling on design Open Questions 1–2, and removing this hold in the same edit (#311). Hold: OPEN-Q proposal awaits owner approval

## 2. Trust check

- [ ] 2.1 `bootstrap.CodexHookTrust(root)` returns the `codex_hook_trust` status and at most one warning (D1, D2, D4). It runs only when `HookSources(root)["codex"] == ".codex/hooks.json"`, reads `$CODEX_HOME/config.toml` or `~/.codex/config.toml` with `go-toml/v2`, and checks `hooks.state` for a key `<abs root>/.codex/hooks.json:<snake_case native event>:` for each of the four `claudeHookEvents`. Doctor reports the status and appends the warning. Premises: `internal/bootstrap/bootstrap.go:739`, `:751`, `:755`; `internal/bootstrap/mcp.go:11`; `cmd/ateam/main.go:615-637`. Acceptance: tests cover every case in the proposal's Definition of Done, and `task mutate:check` reports `detection` for the path match, the per-event check, the missing-config skip, and the user-scope skip (#311).

## 3. Init, docs, and spec

- [ ] 3.1 `ateam init`'s `next_steps` gains the Codex trust sentence (D5). `docs/adoption.md`, in the hooks section and in the T3 Codex row, says to run `codex` once per checkout or worktree and approve the hooks, and that doctor warns when they aren't approved. The `ADDED` Requirement merges into `openspec/specs/harness/spec.md` on archive. Premises: `cmd/ateam/main.go:556`; `docs/adoption.md:36`, `:165`. Acceptance: `task spec:check` and `task test:spec` pass (#311).

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree, `ateam doctor --check` and `ateam bench --selftest` exit 0, and the change is archived before ready (#311).
