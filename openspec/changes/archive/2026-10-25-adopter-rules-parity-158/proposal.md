# Change Proposal: `adopter-rules-parity-158`

## Why

An adopter that keeps its own `AGENTS.md`, `.agents/rules-index.md` and `.agents/protocol.md` gets a permanently red `ateam doctor --check` (#158, from the SemTeams review #157). `checkRulesAndSizeParity` (`internal/bootstrap/bootstrap.go:272`) requires exactly ateam's three section titles from `expectedDomains` (`bootstrap.go:30`), and `checkRequiredFiles` (`bootstrap.go:232`) requires ateam's literal protocol headings `## 1. Hybrid State Architecture` and `## 2. Work Lifecycle & Definition of Done` (`bootstrap.go:250`). `ateam init` correctly preserves the adopter's files, and doctor then rejects them.

## What Changes

- Rules parity means the adopter's two files agree with each other: `AGENTS.md` and `.agents/rules-index.md` have the same numbered rule sections, with the same titles in the same order and the same row count per section. The heading level may differ between the files (`### N.` in one, `## N.` in the other). There is no comparison with `expectedDomains`, which stays as the shipped template's content.
- The protocol is checked for the markers the harness depends on, not for headings: the four Clarify fields (`Deliverable`, `Scope`, `Definition of Done`, `Stop boundary`), `ateam state handoff`, and `openspec/changes/archive/`.
- Diagnostics name the section and file that differ, or the missing obligation and where the upstream template puts it.
- The byte cap on `AGENTS.md` and the `CLAUDE.md` import check are unchanged.

## Non-goals

- A doctor config, allowlist, or blanket parity bypass (#157 asks for the opposite).
- Adopter skills and adapter aliases (#159).
