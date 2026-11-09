## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1 (AGENTS/protocol duplication) and OQ2 (Codex child-run loading) (#200).
  - Evidence: the owner approved in Enjoy on 2026-11-09. OQ1 is (a): the duplication is deferred and recorded in `design.md` Context, not cut here. OQ2 is accepted: the "read it only if it is not in context" wording stands. Claude wrote the header at their request.

## 2. Section-scoped reads

- [x] 2.1 Replace the full-read lines in the five skills and three contracts with section pointers (D1 A, D2 A).
  - Premise: `.agents/skills/{pickup,handoff,preflight,design,review-pr}/SKILL.md:10`; `.agents/contracts/{architect,developer,reviewer}.md:3`.
  - Acceptance: `grep -rn 'protocol.md fully\|protocol.md) first' .agents` prints nothing; `ateam doctor --check` parity is green.
  - Evidence: pickup reads § 1 and § 2 steps 1–2. Handoff reads § 1 and § 2 step 5. In both, an unattended or fan-out run also reads § 4 *Unattended and fan-out runs*. Preflight reads § 2 step 4, § 3 and *Test-to-spec attribution*. Design defers to the architect contract, which reads § 1 and § 2 step 1. Review-pr defers to the reviewer contract. The developer and reviewer contracts read § 1, § 2, § 3, *Test-to-spec attribution* and § 4. § 1 was added after Codex's review of `d100266`: its rule against a tracked `debt-ledger.md` (`protocol.md:21`) appears nowhere else, so a contract run that skipped § 1 lost it. Design D1 had estimated these paths without § 1. The contracts now say `AGENTS.md` is loaded by the host and is read only if it is not in context. The `grep` prints nothing.
- [x] 2.2 Mirror the edits in the adopter templates.
  - Premise: `internal/bootstrap/templates.go:351,368,397,434,482,501`; `internal/bootstrap/review.go:19,30`.
  - Acceptance: `TestAgentSetupParity` and `TestAdopterSkillsAndAliasesPass` pass, and the same `grep` over `internal/bootstrap/*.go` (excluding tests) prints nothing.
  - Evidence: the templates match, except that their pickup and handoff omit the fan-out clause because the template protocol has no such subsection. `go test -race ./internal/bootstrap/` passes and the `grep` prints nothing.
- [x] 2.3 Add a bootstrap test: every `§ n` in a shipped skill or contract template resolves to a `## n.` heading in the protocol template (D3 A).
  - Premise: protocol template headings at `templates.go:280-322`.
  - Acceptance: the test carries `// spec: openspec/specs/harness/spec.md#Requirement: Task-scoped instruction reads`; `task mutate:check` reports `detection` when one template's pointer is changed to `§ 9`.
  - Evidence: `TestTaskScopedInstructionReads` checks the `ateam init` output and this repository. It rejects a full-read line, a `§ n` with no `## n.` heading (a skill's own `skill § n` is exempt), and a named italic section with no heading. `TestReadingViolationsSensitivity` gives the checker one passing and one failing case per rule. Changing the template pickup pointer to `§ 9` makes `task mutate:check` report `detection` (`dangling § 9`). The checker itself lives in a test file, which `ateam mutate` refuses, so the sensitivity table is its only oracle. Codex's review of `d100266` found it missed the old combined opening (`Read [the shared protocol](…) and [the reviewer contract](…) first.`) and a backticked path; restoring the old review-pr opening in the template survived. The checker now flags any sentence that says read … first/fully, names `protocol.md`, and names no `§`. The table covers both forms, the old design opening, and four current pointer sentences that must pass. Restoring the old review-pr opening now gives `detection` (`full read: Read [the shared protocol]`).

## 3. Evidence

- [x] 3.1 Trace each workflow (pickup, small implementation, review, handoff, design) to every obligation it needs, naming where each is read (AGENTS.md, a named section, or the skill).
  - Acceptance: no obligation lives only in an unnamed section. The reviewer re-traces review-pr and handoff.
  - Evidence (A = AGENTS.md, always loaded):
    - Pickup: state homes, drift and holds (§ 1, skill § 1); Clarify (§ 2.1, A); Claim and worktree (§ 2.2, A); writer ownership (skill § 2); lease and fan-out (§ 4 *Unattended*, conditional).
    - Handoff: 5-field checkpoint (§ 1, skill); Close, archive and worktree removal (§ 2.5, A); fan-out handoff slots (§ 4 *Unattended*, conditional).
    - Small implementation: inline `// shortcut:` debt and no tracked debt ledger (§ 1); ladder and footer (A, § 2.3); mutation and PBT (§ 3); attribution (*Test-to-spec*); four DoD gates and draft-CI deferral (§ 2.4); routes and the implemented-by record (§ 4, contract); gate files (§ 4, A).
    - Review: no tracked debt ledger (§ 1); read-only independence and reviewer strength (contract, § 4); record format and merge check (contract, skill § 4–5); mutation, PBT and attribution judgments (§ 3, *Test-to-spec*); gate files (§ 4); DoD (§ 2.4).
    - Design: Clarify, tracks and the one-hold approval (§ 2.1, A); OpenSpec homes (§ 1); file:line and option rules (skill, contract).
    - Read by no workflow: Adoption scope (944 bytes). It concerns adopting a repository, which `docs/adoption.md` covers.
- [x] 3.2 Record before (`3812615`) and after bytes per workflow by `wc -c` at fixed SHAs, separately for static instructions, SessionStart hook injection, MCP schema tokens and board response bytes (D4 A).
  - Evidence: `wc -c AGENTS.md .agents/protocol.md <skill> <contract>` at `3812615`, and `git show 3812615:<file> | wc -c`. Protocol section bytes are unchanged: § 1 2,492; § 2 7,524; § 3 2,766; *Test-to-spec* 3,072; § 4 6,930 (*Unattended* 2,231 of it). AFTER counts whole sections even where a step is named, so it is an upper bound.

    | Workflow | Before | After | Delta |
    | --- | --- | --- | --- |
    | Pickup | 41,720 | 27,884 | −13,836 |
    | Handoff | 41,647 | 27,797 | −13,850 |
    | Small implementation (developer + preflight) | 42,329 | 41,316 | −1,013 |
    | Review (reviewer + review-pr) | 47,938 | 46,869 | −1,069 |
    | Design (architect + design) | 41,723 | 27,871 | −13,852 |

    Before, each contract run could also re-read `AGENTS.md` (+14,881). After, it does so only when the host has not loaded it. An unattended pickup or handoff adds 2,231.
  - Other cost classes, unchanged by this change and measured on this branch's build: SessionStart hook output is 5,360 bytes (it varies with handoff and worktree state); the MCP schema is 257 tokens (`ateam doctor`); `ateam state board --scope open` is 9,106 bytes. No token or productivity gain is claimed.
