# Change Proposal: `openspec-design-rules-218`

Approved-by: Coby Leuschke (2026-10-30)

## Why

`ateam init` writes `openspec/config.yaml` as `schema: spec-driven` only (`internal/bootstrap/templates.go:593`), while this repository's config carries `rules: design:`. So in an adopter, `openspec instructions design` shows none of the design rules, which only `.agents/skills/design/SKILL.md` carries (#218).

## What Changes

1. `ateam init` writes `openspec/config.yaml` as `schema: spec-driven` plus this repository's six `rules: design:` lines (constant `openspecConfig`). An existing `openspec/config.yaml` stays untouched, as before. The ateam-specific `context:` block is not shipped.
2. A test keeps the generated `rules:` block identical to this repository's `openspec/config.yaml`.
3. `docs/adoption.md` and `docs/brownfield.md` describe the new output and drop the "(#218)" deferral.
4. Spec: a scenario added to the harness Requirement "Opt-in design track".

## Track

`quick`. One package (`internal/bootstrap`) and two docs. It adds a scenario but no Requirement, adds no exported surface, and touches no boot path, durability behavior, or gate file. Adopter-visible effect: freshly initialized repositories get the rules; existing configs do not change.

## Non-goals

- Doctor checking the rules in an adopter's config, or migrating existing configs.
- Removing the duplicated text from the design skill; Codex and Gemini read the skill without OpenSpec.

## Definition of Done

- `TestInitWritesDesignRulesMatchingThisRepository` passes, and `task mutate:check` reports `detection` when one template rule is edited.
- `openspec validate --all --strict` passes on a freshly initialized repository, and `openspec instructions design` there lists the rules.
- `task verify` is green on a clean tree, `ateam doctor` parity is green, and `ateam bench --selftest` is green.
