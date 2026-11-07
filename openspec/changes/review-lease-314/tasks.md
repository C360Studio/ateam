## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 2 (#314, #315). Evidence: the owner approved in Enjoy on 2026-11-07 ("Approved, with question 2 as recommended"), so OQ2 is no and the stale-MCP marker stays per session. Claude wrote the header at their request.

## 2. Lease

- [x] 2.1 `HandleWithAgent` reads `ATEAM_ROLE == "review"` and `handle` skips the three `WriteWriterLease` calls for it (D1 A). Premises: `internal/hook/hook.go:89-91`, `:145`, `:159`, `:277`. Acceptance: a hook test seeds another session's lease, runs session-start, before-prompt, and turn-end with `ATEAM_ROLE=review`, and asserts the lease file is byte-for-byte unchanged. Without the variable, and with `Review`, the same events write the lease. `task mutate:check` reports `detection` with each guard removed and with the comparison made case-insensitive (#315). Evidence: `internal/hook/hook.go:92` reads the role. Session-start's guard is at `:154`. Before-prompt (`:169`) and turn-end (`:287`) call `refreshWriterLease` (`:104`), which was split out to keep `handle` within revive's function-length limit. `TestReviewRoleWritesNoWriterLease` covers another writer's lease kept byte-for-byte by each event, no lease taken in an empty checkout by each event, and the roles `""`, `Review`, and `reviewer` each writing the lease on all three events. `go run ./cmd/ateam mutate` reports `detection` for each mutant: the session-start guard removed, the helper's guard removed, `false` passed at each of the two call sites, and `strings.EqualFold`.

## 3. Launches

- [ ] 3.1 `.agents/hosts/t3code.toml:15` creates the detached review checkout and passes `workspaceStrategy {type: existing_worktree}` for it (D2), after Open Question 1 is checked in one live T3 Codex review: the thread's cwd is the checkout, and `main`'s lease is unchanged. Acceptance: `TestReviewRequest` still expands the manifest, and the live-run evidence is recorded here (#314). Progress: the manifest is written with `worktreePath` (design OQ1), and `ATEAM_HOST=t3code ateam review request --dry-run` expands it. The live run is not done yet.
- [x] 3.2 `.agents/hosts/herdr.toml:12` adds `--env ATEAM_ROLE=review` to `herdr pane split` (D3). Acceptance: `TestReviewRequest` expands it (#315). Evidence: `TestReviewRequest` passes, and `ATEAM_HOST=herdr ateam review request --dry-run` prints `herdr pane split --pane X --direction right --no-focus --env ATEAM_ROLE=review`.
- [x] 3.3 `.agents/skills/review-pr/SKILL.md:22` reuses a launch-created checkout (D4); `docs/adoption.md` states the terminal form and the Enjoy and T3 naming-run limits (D5). Acceptance: `task spec:check` and `ateam doctor --check` pass (#314, #315). Evidence: the shipped copy in `internal/bootstrap/templates.go` is updated so the parity test passes. The paragraph sits above the `{return_to}` refusal in `docs/adoption.md`.

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree, `ateam doctor --check` and `ateam bench --selftest` exit 0, the owner has applied `gate-change-approved`, and the change is archived before ready (#314, #315).
