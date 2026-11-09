## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by:` with the digest `spec:queue` prints and removing the `OPEN-Q` line in the same edit, ruling on design Open Questions 1–3 (#359). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 `gateSource` gains `headCheck(ctx, sha, name string) (passed bool, err error)`: GitHub `repos/{owner}/{repo}/commits/<sha>/check-runs?check_name=<name>` (latest run by `completed_at`, `conclusion == success`); Gitea `repos/{owner}/{repo}/commits/<sha>/statuses`, the newest status whose context is the `Merge check` job's, `state == success`. Exact name match only. Acceptance: fake-forge tests for success, failure, pending, absent, a `Merge check probe` decoy, and a read error on each forge (#359).
- [ ] 2.2 `awaiting()` (`internal/review/gate.go:321-357`): after the documents-only and line checks, replace the record loop with `headCheck(pull.Head, "Merge check")`; a pass answers `false`, anything else `true`; the doc comment states why. Acceptance: the `TestGateAwaiting` rows in `proposal.md` § Definition of Done replace the three record rows (`gate_test.go:410-416`); `task mutate:check` `detection` for the inverted pass test and a prefix match (#359).
- [ ] 2.3 `.github/workflows/merge-check.yml:34-37` adds `checks: read` to the `awaiting` job and updates the probe comment (`:24-28`); `.gitea/workflows/merge-check.yml` updates its comment, and its scope per Open Question 2. Acceptance: a live run on this PR shows `Merge check (awaiting review record, skipped)` after a record at the head with no line, then `Merge check` green after the line is written (#359).
- [ ] 2.4 `specs/harness/spec.md` delta: MODIFIED Requirement *Merge check runs on pull-request state events* (`spec.md:820-867`), drafted for D1 A in the awaiting, probe, and line-removal scenarios; rewrite those three if the owner rules D1 B. Acceptance: `task spec:check` and `task test:spec` pass (#359).

## 3. Verify

- [ ] 3.1 `task verify` green on a clean tree; `ateam doctor --check` and `ateam bench --selftest` exit 0; owner `gate-change-approved` at the final head; change archived before ready (#359).
