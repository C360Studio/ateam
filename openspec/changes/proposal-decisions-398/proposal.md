# Proposal: proposal-decisions-398
Approved-by:

## Track

`design`: adds a harness Requirement, changes `spec-check` behaviour adopters run, and changes generated documents (protocol, design skill, architect contract, `openspec/config.yaml`).

## Decisions you are making by approving

- When a proposal has no decisions section and the change has a `design.md`, `task spec:check` fails; without a `design.md` it only warns.
- A decisions bullet made mostly of code names or numbers draws a warning asking for plain words.
- When a spec Requirement in a change grows past 500 characters, you hear about it while the change is open, not after archive, whatever OpenSpec version is installed.
- When a single scenario line runs past 60 words, `spec:check` warns; it never fails for length.
- Changes already archived are never checked; an open change elsewhere (today #406) must add the section to pass once it is on the design track.

## Why

An adopter's owner approved behaviour they did not know they were deciding (#398): the decisions sat inside 150–250-word Requirement sentences, and nothing in `proposal.md` said in plain language what would change. Two were safety-relevant defaults the owner reversed once stated plainly.

## What Changes

1. `spec-check` looks for `## Decisions you are making by approving` in each open change's `proposal.md` holding at least one bullet; missing fails with a `design.md`, warns without.
2. `spec-check` warns on a decisions bullet with fewer than 5 words outside code spans.
3. `spec-check` warns on a delta Requirement body over 500 characters (ADDED and MODIFIED) and a scenario line over 60 words.
4. Protocol § 2 step 1, the `design` skill, the architect contract and a `rules: proposal:` block in `openspec/config.yaml` ask for the section; each lands in its repository copy and its `ateam init` template.

## Scope & Non-Goals

- **In-bounds:** `internal/state/openspec.go` and tests, `openspec/specs/harness/spec.md` (one ADDED Requirement), `.agents/protocol.md`, `.agents/skills/design/SKILL.md`, `.agents/contracts/architect.md`, `openspec/config.yaml`, their templates in `internal/bootstrap/templates.go`.
- **Non-goals:** judging whether a bullet is honest or complete (#398 *Limits*), archived changes, a `## Track` parser, Requirement length in canonical specs (OpenSpec already fails it).

## Definition of Done

- Table tests with literal proposals cover: section missing with and without `design.md`, empty section, code-only bullet, 499/501-char Requirement, 60/61-word scenario line; `task mutate:check` detects removal of each rule.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
