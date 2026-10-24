## 1. Implementation

- [ ] 1.1 `checkRulesAndSizeParity` compares the adopter's `AGENTS.md` and `.agents/rules-index.md` with each other: the same numbered sections, titles and order (heading level may differ) and the same row count per section; no comparison with `expectedDomains`; the byte cap and `CLAUDE.md` import check are unchanged (#158).
- [ ] 1.2 `checkRequiredFiles` checks `.agents/protocol.md` for the Clarify fields, `ateam state handoff` and `openspec/changes/archive/` instead of two literal headings; each missing marker is a named violation that says where the upstream template puts it (#158).
- [ ] 1.3 Tests: a brownfield fixture in `internal/bootstrap` (five project-named sections in both files, a protocol with different headings that carries the markers, hooks and MCP configured) passes `doctor --check`; a title differing between the files fails naming the section; a row-count mismatch fails; a protocol without the Clarify fields fails; the ateam repo's own files still pass. `task mutate:check`: dropping the title comparison must be `detection`.
- [ ] 1.4 Move the requirement into `openspec/specs/harness/spec.md` and cite it there, then archive the change.
