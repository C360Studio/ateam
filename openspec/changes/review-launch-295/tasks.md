## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1 and 2 (#295). Ruled 2026-11-06: Q1 `agent` for this repository; Q2 yes, `.agents/hosts/` is a gate file.

## 2. Policy and request

- [ ] 2.1 `review_launch` on `Policy` (`internal/review/policy.go:26`), validated beside `review_record` (`:120`); `ateam doctor` accepts a policy with it (#295). Acceptance: tests for absent, `manual`, `agent`, invalid; `task mutate:check` reports `detection` with the invalid branch removed.
- [ ] 2.2 `Policy.Eligible`-style candidate filter beside `Compare` (`internal/review/compare.go:28`): other-provider models not below each implementer's tier, with the highest implementer band (#295). Acceptance: tests for a frontier implementer, a workhorse implementer, two implementers, and `cross_provider: false` (same provider allowed); `detection` with the tier comparison inverted.
- [ ] 2.3 `ateam review request <pr>` in `cmd/ateam/review.go:21` printing the D2 object, reading implementers from flags or the PR body, executing only under `agent` with a manifest `launch`, no shell (#295). Acceptance: a test per `cross_provider` and `review_launch` pair, offline-without-flags exits 2, a recorder runner proves the argument array and that `manual` never runs it.

## 3. Host manifest and notify

- [ ] 3.1 `.agents/hosts/<host>.toml` parser and placeholder expander (D3), host detection (D4), `--return-to` (#295). Acceptance: tests for every placeholder, a placeholder inside one argument, a missing manifest, a malformed manifest (fixed-text error), the detection order and `ATEAM_HOST`; `detection` with the order swapped and with the expander skipping one placeholder.
- [ ] 3.2 `ateam review notify` sending the D6 text through `notify`, printing the host tool on an MCP-only host, nothing on `cli` (#295). Acceptance: a test with a record body holding shell text and the warning line proves the message is the fixed text; `detection` with the fixed text replaced by the body.
- [ ] 3.3 Reference manifests `.agents/hosts/enjoy.toml` and `.agents/hosts/herdr.toml`, each verified against the host once by hand and the evidence recorded here (#295). Acceptance: `ateam review request --dry-run` prints the expanded arguments for each.

## 4. Record in the cache and pickup

- [ ] 4.1 `review.ParseRecordComment` and the gate (`internal/review/gate.go:421-441`) using it (#295). Acceptance: gate tests unchanged and green; parser tests for a valid record, missing sha, missing verdict, two records.
- [ ] 4.2 `sync` fetches comments for worktree-backed open PRs under `ghCommentBudget` (`internal/state/forge.go:501`, Gitea `:316`) and fills `BoardCard.Review` (#295). Acceptance: GitHub and Gitea fixture tests; a budget-exhausted test leaves `Review` empty and sync still exits 0.
- [ ] 4.3 Pickup worktree line (`internal/state/state.go:2124`, `:2998`) prints the record and `stale` (#295). Acceptance: tests for present, absent, stale; `detection` with the stale comparison inverted; `ateam doctor` hook latency before and after.

## 5. Prose and verify

- [ ] 5.1 Protocol § Close (`.agents/protocol.md:45`), review-pr § 4 (`.agents/skills/review-pr/SKILL.md:24`) and handoff skills, `internal/bootstrap` templates and drift tests, `docs/adoption.md` hosts section after `:119`, ADR 0002 addendum after `:131` (#295). Acceptance: `ateam doctor --check` and `task spec:check` pass; the adoption section names no adopter.
- [ ] 5.2 `task verify` green on a clean tree; `ateam doctor --check` and `ateam bench --selftest` exit 0; this repository's policy set per Open Question 1; change archived before ready (#295).
