## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#178).

## 2. Implementation

- [ ] 2.1 Protocol track text (#178). Premise: `.agents/protocol.md:30` Stop boundary offers "pause after design/plan" undefined; `:18` names `design.md` once. Check: § 2.1 lists `quick`/`design`, the four triggers, and that one owner approval covers both files; the `ateam init` protocol mirror in `internal/bootstrap/templates.go` matches.
- [ ] 2.2 `design` rules in `openspec/config.yaml` (#178). Premise: `openspec/config.yaml` has `schema` and `context` only. Check: `openspec instructions design` prints the rule text; `openspec validate --all --strict` still passes.
- [ ] 2.3 Canonical skill and adapters (#178). Premise: `canonicalSkills` at `internal/bootstrap/bootstrap.go:30` lists five; `requiredSkillMarkers` at `:431`. Check: `.agents/skills/design/SKILL.md`, `.claude/skills/design/SKILL.md`, `.gemini/commands/design.toml` exist; init template writes them; a bootstrap test covers init output and doctor failure on a removed canonical file.
- [ ] 2.4 Design length warning (#178). Premise: `maxProposalLines` and the warning at `internal/state/openspec.go:51`, `:484`. Check: `TestDesignLengthWarning` at 120, 121, and absent; mutation on the comparison and the limit each `detection`.
- [ ] 2.5 Architect contract (#178). Premise: `.agents/contracts/` holds `developer.md`, `reviewer.md`. Check: `architect.md` states read-only, drafts, never implements, never rules.
- [ ] 2.6 Rules rows and spec (#178). Check: one `review` row in `AGENTS.md` § 3 and `.agents/rules-index.md` with matching counts (`ateam doctor`); `AGENTS.md` under 24,000 bytes; harness Requirement "Opt-in design track" scenarios pass `task test:spec` attribution.
