## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#198). Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Rescope the rule in `AGENTS.md:13,64`, `.agents/rules-index.md:15`, and `.agents/protocol.md:43` Gate 4 (#198). Acceptance: the row text matches in both tables; `ateam doctor --check` row parity passes.
- [ ] 2.2 Rescope `.agents/skills/ponytail/SKILL.md:5,11`, `.agents/skills/preflight/SKILL.md:31`, and `.agents/skills/review-pr/SKILL.md:35` (#198). Acceptance: each names a conclusion, not every reply.
- [ ] 2.3 Mirror 2.1–2.2 into `internal/bootstrap/templates.go:183,204,245,436,442,474,510` and `internal/bootstrap/reviewrecord.go:47` (#198). Acceptance: the existing bootstrap parity tests pass with canonical and template text byte-identical.
- [ ] 2.4 Change the last line of `LadderPrompt` in `internal/hook/hook.go:430` to "End conclusions and handoffs with 1-2 lines" (#198). Acceptance: `task test:unit` green; doctor hook latency within budget.
- [ ] 2.5 Verify (#198). Acceptance: the `grep` in the proposal's Definition of Done is empty; `task verify` green on a clean tree.
