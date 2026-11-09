## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1 (AGENTS/protocol duplication) and OQ2 (Codex child-run loading) (#200).
  - Hold: OPEN-Q proposal awaits owner approval

## 2. Section-scoped reads

- [ ] 2.1 Replace the full-read lines in the five skills and three contracts with section pointers (D1 A, D2 A).
  - Premise: `.agents/skills/{pickup,handoff,preflight,design,review-pr}/SKILL.md:10`; `.agents/contracts/{architect,developer,reviewer}.md:3`.
  - Acceptance: `grep -rn 'protocol.md fully\|protocol.md) first' .agents` prints nothing; `ateam doctor --check` parity is green.
- [ ] 2.2 Mirror the edits in the adopter templates.
  - Premise: `internal/bootstrap/templates.go:351,368,397,434,482,501`; `internal/bootstrap/review.go:19,30`.
  - Acceptance: `TestAgentSetupParity` and `TestAdopterSkillsAndAliasesPass` pass, and the same `grep` over `internal/bootstrap/*.go` (excluding tests) prints nothing.
- [ ] 2.3 Add a bootstrap test: every `§ n` in a shipped skill or contract template resolves to a `## n.` heading in the protocol template (D3 A).
  - Premise: protocol template headings at `templates.go:280-322`.
  - Acceptance: the test carries `// spec: openspec/specs/harness/spec.md#Requirement: Task-scoped instruction reads`; `task mutate:check` reports `detection` when one template's pointer is changed to `§ 9`.

## 3. Evidence

- [ ] 3.1 Trace each workflow (pickup, small implementation, review, handoff, design) to every obligation it needs, naming where each is read (AGENTS.md, a named section, or the skill).
  - Acceptance: no obligation lives only in an unnamed section. The reviewer re-traces review-pr and handoff.
- [ ] 3.2 Record before (`3812615`) and after bytes per workflow by `wc -c` at fixed SHAs, separately for static instructions, SessionStart hook injection, MCP schema tokens and board response bytes (D4 A).
