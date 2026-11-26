# Tasks: coordinator-417

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-26) sha256:bf0ac7bdbf7e`), written by claude on the owner's instruction; rulings D1 A, D2 A, D3 A, D4 A, D5 A.

## 2. Implementation

- [ ] 2.1 Contract (premise `.agents/contracts/architect.md:1-9` shape, `docs/adr/0003-roles-and-routes.md:24-30`): write `.agents/contracts/coordinator.md` covering the write scope from D3, the stop list (every owner gate: proposal approval, `gate-change-approved`, `status:needs-decision`, a third REQUEST CHANGES, milestone scope or order, release, red `main`), batching, and the D2 worktree. Accept: review against this design's D2–D5.
- [ ] 2.2 Skill (premise `.agents/skills/pickup/SKILL.md` numbered-heading shape, `.agents/hosts/*.toml` `[launch]`): write `.agents/skills/coordinate/SKILL.md` with the loop (`ateam state next`, `queue-check`, developer launch per D4, `ateam review request`, relay, gate wait, batched prompt, `owner-prompts:`), plus `.claude/skills/coordinate/SKILL.md` and `.gemini/commands/coordinate.toml`. Accept: `ateam doctor --check` green with `coordinate` treated as canonical.
- [ ] 2.3 Bootstrap (premise `internal/bootstrap/bootstrap.go:30`, `internal/bootstrap/templates.go:607-611`): add `coordinate` to `canonicalSkills`, and add the contract and skill bodies to the templates. Accept: an init test into an empty directory finds the contract, the skill and both adapters with the expected link text; a mutation that drops the contract template is detected.
- [ ] 2.4 Docs (premise `docs/adr/0003-roles-and-routes.md:24-41`, `openspec/specs/harness/spec.md:606`): add the ADR 0003 catalog and route rows (`frontier`/`high` with no `workhorse` exception, since it routes and never implements); `.agents/README.md`; `docs/daily-use.md`; and the spec delta. Accept: `task spec:check` green.
- [ ] 2.5 Dogfood: one coordinated run on this repository that merges at least one PR carrying `owner-prompts:`. Record the count on #417. Accept: the merged PR body.
- [ ] 2.6 Run `task verify`, `ateam doctor --check` and `ateam bench --selftest`, then archive this change.
