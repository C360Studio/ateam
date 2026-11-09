## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Question 1 (#325, #322). Hold: OPEN-Q proposal awaits owner approval

## 2. Change

- [x] 2.1 `gateIsDoc` treats `.agents/skills/review-pr/` as code. The code-PR definition in `AGENTS.md`, `.agents/rules-index.md`, `internal/bootstrap/templates.go`, `docs/model-review.md` and the harness spec delta names it (#325). Acceptance: the `TestGateGitea` case `the review-pr skill is code`. Evidence: `task mutate:check` with the clause joined by `||` reports `detection` (`code file: .agents/skills/review-pr/SKILL.md`).
- [x] 2.2 The record status reads `<VERDICT> self-reported by …` and the passing merge-check line names the self-reported limit. `docs/adoption.md` shows the status (#322). Acceptance: `TestChooseStatusesD3Table` and the `TestGateGitea` `record at head` case. Evidence: both mutants (the old status text, and the summary without the clause) report `detection`.

## 3. Verify

- [ ] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#325, #322).
