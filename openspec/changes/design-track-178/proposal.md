# Change Proposal: `design-track-178`

Approved-by:

## Why

Planning depth is undeclared. A non-trivial change carries `design.md` only if the agent felt like writing one: 19 of 66 archived changes have one, 9 of those are 9 lines or fewer, and nothing reads them (`ScanOpenSpecQueue` and `CheckOpenSpec` in `internal/state/openspec.go` read `tasks.md` and `proposal.md` only). The Clarify gate offers "pause after design/plan" without saying what a design is (#178, epic #180). #142 added the approval hold; this change defines what is approved on the deeper track.

## What Changes

- **Track at Clarify.** `.agents/protocol.md` § 2.1: a non-trivial change is `quick` or `design`. `design` is required when the change adds a spec Requirement, touches more than one capability under `openspec/specs/`, adds exported surface reachable from an adopting repository, or touches a boot path, durability/ack behavior, or a gate file. Otherwise `quick`, with no `design.md`. On `design` the one #142 `OPEN-Q` hold covers proposal and design; one owner edit lifts both.
- **Shape is OpenSpec's.** `design.md` keeps the `spec-driven` sections (Context, Goals / Non-Goals, Decisions, Risks / Trade-offs, Migration Plan, Open Questions). ateam adds, as `rules:` for the `design` artifact in `openspec/config.yaml`: Context is `file:line` claims opened, not remembered; each Decision names at least two options with cost, blast radius, what it forecloses, and the adopter seam; owner-ruling questions are marked; each task in `tasks.md` carries its `file:line` premise and acceptance check.
- **Canonical skill `design`.** `.agents/skills/design/SKILL.md` with `.claude/skills/design/SKILL.md` and `.gemini/commands/design.toml`; read-only, drafts and never rules. `canonicalSkills` grows to six with its marker entry and `ateam init` template (`internal/bootstrap`).
- **Length warning.** `task spec:check` warns, exit 0, when `design.md` exceeds 120 lines, as it does for `proposal.md` past 60.
- **Architect contract.** `.agents/contracts/architect.md` beside `developer.md` and `reviewer.md`: read-only, drafts the design, never implements, never rules.
- **Rows.** One `review` row in `AGENTS.md` § 3 and `.agents/rules-index.md`, mirrored in the `ateam init` templates; one harness spec Requirement.

## Non-goals

- Model routing per role (#179, ADR 0003) and any `routes` policy change (gate file).
- A new Ponytail mode, a parsed `Track:` field, or detecting triggers from a diff: the trigger list is protocol text checked at review.
- PRD, UX, or other per-change artifacts; wiring OpenSpec's `/opsx:*` slash skills.
- Gate files: nothing under `internal/review/`, `scripts/merge-check.sh`, or the workflows.
- This change does not take the design track itself: it is the bootstrap. The next change that meets a trigger is the first to run it end to end.

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor --check` parity green with six skills.
- Tests: `TestDesignLengthWarning` (120 no warning, 121 warning, absent `design.md` no warning) in `internal/state/openspec_test.go`; a bootstrap test that `ateam init` writes the `design` skill and both adapters and that doctor fails when the canonical file is removed; each shown able to fail by `task mutate:check` or `go test -overlay` with SHA-256 recorded.
- `openspec instructions design` shows the new `rules:` text.
