# Change Proposal: `adopter-skills-159`

## Why

Doctor treats every directory under `.agents/skills/` as a generated ateam skill, so an adopter's own skills and its existing adapter aliases produce violations that an unchanged upstream binary cannot clear (#159, from the SemTeams review #157). `canonicalSkills` (`internal/bootstrap/bootstrap.go:28`) lists the five generated skills; `discoverSkills` (`:426`) appends every other directory to the same list, and `checkSkillAdapters` (`:452`) then demands a `## 1.` heading, procedure markers, same-name Claude and Gemini adapters, link text, a line cap and no checkboxes for each one. SemTeams' `query-pattern` skill has prose headings and no Gemini adapter, and its `.claude/skills/preflight/` deliberately links to `.agents/skills/semteams-preflight/`.

## What Changes

- The strict checks apply to the five canonical skills only, exactly as today.
- Adopter skills are checked for one thing: an adapter (`.claude/skills/<x>/SKILL.md` or `.gemini/commands/<x>.toml`) that links into `.agents/skills/` must point at a file that exists. Doctor lists them under `adopter_skills` in its JSON.
- An alias is an adapter whose link target names a different skill. It passes when the target exists and is reported as `alias: .claude/skills/<x> -> .agents/skills/<y>`. When a canonical skill's adapter aliases elsewhere, the canonical checks read the aliased target instead of the generic file.

## Non-goals

- A plugin framework or a skills manifest (#157 does not ask for one).
- Rules and protocol parity (#158, landed).
