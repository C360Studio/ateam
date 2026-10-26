## ADDED Requirements

### Requirement: Adopter skills and adapter aliases

`ateam doctor` SHALL apply its canonical-skill checks only to the five skills ateam generates (`pickup`, `handoff`, `preflight`, `ponytail`, `review-pr`), and SHALL check other skills and adapters only for links that resolve.

#### Scenario: Adopter skill

- **WHEN** `.agents/skills/` holds a skill that is not one of the five canonical skills and that no canonical adapter aliases
- **THEN** doctor requires no numbered heading, procedure markers, adapters, line cap or checkbox rule for it, and lists it under `adopter_skills` in its JSON output (an empty list when there is none)

#### Scenario: Dangling adapter link

- **WHEN** a `.claude/skills/<x>/SKILL.md` or `.gemini/commands/<x>.toml` adapter links to a path under `.agents/skills/` that is not a regular file (a link naming only a skill means its `SKILL.md`; a link starting `./` or `../` resolves from the adapter's own directory, a bare one from the repository root)
- **THEN** doctor reports a violation naming the adapter and the missing target (`<adapter>: links to missing <target>`), and does not report the link as an alias

#### Scenario: Adapter alias

- **WHEN** an adapter named `<x>` links to an existing `.agents/skills/<y>/SKILL.md` with `<y>` different from `<x>` (an adapter's skill is the one its first `.agents/skills/` link names, and a trailing `.` is not part of the name)
- **THEN** doctor lists `alias: .claude/skills/<x> -> .agents/skills/<y>` (or `alias: .gemini/commands/<x>.toml -> .agents/skills/<y>`) under `skill_aliases` in its JSON output without a violation, and when `<x>` is a canonical skill its canonical checks read `.agents/skills/<y>/SKILL.md` and also `.agents/skills/<x>/SKILL.md` (Codex and Antigravity read `.agents/skills/` directly, so the generic file is checked even when both adapters alias elsewhere), and its adapter must link to `<y>` and say "fully"

#### Scenario: Canonical adapters resolving to different skills

- **WHEN** the Claude and Gemini adapters of a canonical skill resolve to different skills, for example the Claude adapter aliases `<y>` and the Gemini adapter links to the generic `<x>`
- **THEN** doctor applies that canonical skill's checks to each resolved `SKILL.md` and to the generic `.agents/skills/<x>/SKILL.md`, and a canonical adapter that is missing or has no `.agents/skills/` link must link to the generic file as before
