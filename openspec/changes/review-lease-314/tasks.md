## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 2 (#314, #315). Hold: OPEN-Q proposal awaits owner approval

## 2. Lease

- [ ] 2.1 `HandleWithAgent` reads `ATEAM_ROLE == "review"` and `handle` skips the three `WriteWriterLease` calls for it (D1 A). Premises: `internal/hook/hook.go:89-91`, `:145`, `:159`, `:277`. Acceptance: a hook test seeds another session's lease, runs session-start, before-prompt, and turn-end with `ATEAM_ROLE=review`, and asserts the lease file is byte-for-byte unchanged. Without the variable, and with `Review`, the same events write the lease. `task mutate:check` reports `detection` with each guard removed and with the comparison made case-insensitive (#315).

## 3. Launches

- [ ] 3.1 `.agents/hosts/t3code.toml:15` creates the detached review checkout and passes `workspaceStrategy {type: existing_worktree}` for it (D2), after Open Question 1 is checked in one live T3 Codex review: the thread's cwd is the checkout, and `main`'s lease is unchanged. Acceptance: `TestReviewRequest` still expands the manifest, and the live-run evidence is recorded here (#314).
- [ ] 3.2 `.agents/hosts/herdr.toml:12` adds `--env ATEAM_ROLE=review` to `herdr pane split` (D3). Acceptance: `TestReviewRequest` expands it (#315).
- [ ] 3.3 `.agents/skills/review-pr/SKILL.md:22` reuses a launch-created checkout (D4); `docs/adoption.md` states the terminal form and the Enjoy and T3 naming-run limits (D5). Acceptance: `task spec:check` and `ateam doctor --check` pass (#314, #315).

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree, `ateam doctor --check` and `ateam bench --selftest` exit 0, the owner has applied `gate-change-approved`, and the change is archived before ready (#314, #315).
