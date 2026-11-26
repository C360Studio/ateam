# Proposal: coordinator-417
Approved-by: Coby Leuschke (2026-11-26) sha256:bf0ac7bdbf7e

## Track

`design`: adds a harness Requirement, a new contract and skill that `ateam init` ships to adopting repositories, and a row in ADR 0003's role catalog. This is the first of two changes for #417 (owner ruling 2026-11-26). Hold-my-beer, where a judge stands in for the owner, is the second change (#424).

## Decisions you are making by approving

- ateam gains a coordinator role. It picks the next issue, starts the developer and reviewer runs, passes review results between them, and asks you when it reaches a decision only you can make. It never writes code, never reviews, and never fills in your approvals.
- The coordinator stops at every decision that is yours: approving a proposal, approving a gate-file change, an issue marked as needing a decision, a third rejected review, milestone scope or order, and a release. It gathers the questions that are waiting into one message instead of interrupting you once per question.
- The coordinator reports queue problems, such as a stale blocked label or an issue with no milestone, and leaves the fix to you. It changes no labels or milestones itself.
- A coordinator works from a worktree and branch of its own, so its handoff never collides with a developer's on `main`.
- The coordinator starts developer runs with the same host tools that already start reviewers (Enjoy, herdr, T3 Code). When the host cannot start one, it gives you the prompt to start it yourself, and that counts as one question to you.
- Each pull request the coordinator moves through records how many times it had to ask you. That count is the baseline hold-my-beer will be measured against.
- Every repository that runs `ateam init` gets the coordinator contract and skill. They do nothing until someone asks a session to coordinate.

## Why

ateam lists architect, developer, reviewer and judge (ADR 0003), but nobody owns the work between them. Choosing the next unit, starting runs, relaying reviews and cleaning up all fall to the owner or to whichever session happens to be open. During v0.4.5 two sessions worked from different handoffs on `claude@main`. Before a judge stands in for the owner, the number of owner prompts per merged PR needs measuring.

## What Changes

1. `.agents/contracts/coordinator.md`: what the role may write (its handoff, PR comments, and the `owner-prompts:` line in a PR body; archiving, arming auto-merge and cleanup stay with the developer run that claimed the work), what it never does, and when it stops.
2. `.agents/skills/coordinate/SKILL.md` with its Claude and Gemini adapters: the loop is `ateam state next` and `queue-check`, then launch, relay, gate wait, and a batched owner prompt.
3. `internal/bootstrap`: the contract and skill join the files `ateam init` writes, and `coordinate` joins the canonical skills.
4. ADR 0003 § 1 gets a `coordinator` row, and § 2 gets its default route.
5. `openspec/specs/harness/spec.md` gets one ADDED Requirement.

## Scope & Non-Goals

- **In-bounds:** the files above, plus `.agents/README.md`, `AGENTS.md`/`.agents/rules-index.md` if a rule row is needed, and `docs/daily-use.md`.
- **Non-goals:** judges, `Judged-by:`, the policy opt-in, and the ADR 0003 judge amendment, all in #424. Out of scope here: any edit to a gate file, including `.agents/hosts/`; a scheduler; a new ateam runtime or command; changing labels or milestones.

## Definition of Done

- Bootstrap tests show `ateam init` writes the contract, the skill and both adapters, and `ateam doctor --check` applies the canonical-skill checks to `coordinate`.
- One coordinated run on this repository merges at least one PR, and that PR body carries `owner-prompts:`.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before ready.
