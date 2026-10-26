## ADDED Requirements

### Requirement: Adopter skills and adapter aliases

`ateam doctor` SHALL apply its canonical-skill checks only to the five skills ateam generates (`pickup`, `handoff`, `preflight`, `ponytail`, `review-pr`), and SHALL check other skills and adapters only for links that resolve.

#### Scenario: Adopter skill

- **WHEN** `.agents/skills/` holds a skill that is not one of the five canonical skills
- **THEN** doctor requires no numbered heading, procedure markers, adapters, line cap or checkbox rule for it, and lists it under `adopter_skills` in its JSON output

#### Scenario: Dangling adapter link

- **WHEN** a `.claude/skills/<x>/SKILL.md` or `.gemini/commands/<x>.toml` adapter links to a file under `.agents/skills/` that does not exist
- **THEN** doctor reports a violation naming the adapter and the missing target

#### Scenario: Adapter alias

- **WHEN** an adapter named `<x>` links to an existing `.agents/skills/<y>/SKILL.md` with `<y>` different from `<x>`
- **THEN** doctor reports `alias: .claude/skills/<x> -> .agents/skills/<y>` without a violation, and when `<x>` is a canonical skill its canonical checks read `.agents/skills/<y>/SKILL.md` instead of `.agents/skills/<x>/SKILL.md`
