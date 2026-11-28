# ADR 0003: Role catalog and default routes

- **Status:** Accepted (owner, 2026-10-27); amended 2026-11-26 to add the `coordinator` row (#417) and the hold-my-beer judge (#424); amended 2026-11-28 to record the judge's developer-question and status-label rulings (#432, #443) and the contracts `ateam init` ships
- **Date:** 2026-10-27
- **Deciders:** repository owner
- **Builds on:** [ADR 0001](0001-review-gate-scope.md) (routes are attestations), [ADR 0002](0002-model-tiers-and-effort-bands.md) (tiers, bands, the not-weaker rule)
- **Evidence:** #179, epic #180, #178 (design track and architect contract, PR #212), PR #214 (live gate separated from the optional route path)

## Context

This repository documents two roles as contracts, developer and reviewer (`.agents/contracts/developer.md`, `reviewer.md`), and ranks runs by model tier then effort band (ADR 0002). #178 added a third contract, `architect.md`, for the design track. Three things are not written down anywhere in the repository:

1. **The catalog.** Which roles exist, what each may write, and what each may never do.
2. **Planning and judgment roles.** The architect contract exists, but no document says what tier drafts a design or who rules on a question the architect marks for the owner.
3. **Default routes per task class.** Which tier implements what, when a `workhorse` developer is acceptable, and when a judgment goes to the top tier. That text lives in the owner's private instructions and global agent frontmatter, outside the repository.

`review-policy.json` `routes` knows only developer and reviewer (`internal/review/policy.go:107-110`), and this repository leaves it unconfigured: `ateam review route` exits non-zero by design (protocol § 4).

## Decision

### 1. Catalog

| Role | Contract | May write | Never |
| --- | --- | --- | --- |
| `architect` | `.agents/contracts/architect.md` | The change's `design.md` and the premises in its `tasks.md` | Implement, rule, or edit any other file |
| `developer` | `.agents/contracts/developer.md` | The complete change: code, callers, tests, fixtures, OpenSpec task evidence; on the owner's yes naming its PR and head, the `gate-change-approved` comment and label (#453) | Review its own work, or lower its model to make review possible |
| `reviewer` | `.agents/contracts/reviewer.md` | One review record comment on the pull request | Edit the deliverable it grades |
| `coordinator` | `.agents/contracts/coordinator.md` | Its handoff, PR comments relaying review results or the owner's batched questions, and a PR body's `owner-prompts:` line | Implement, review, rule, fill `Approved-by:`, apply `gate-change-approved`, change labels or milestones, or release |
| `judge` | `.agents/contracts/judge.md` (#424); otherwise the owner's prompt states the one question | A ruling on that question, posted where the owner asks; in hold-my-beer mode, a proposal's `Judged-by:` line with its hold removed, a milestone's `Order:` line or an issue's milestone, and removal of the `status:needs-decision` or `status:blocked` label it was asked about | Sweep, implement, rule on anything it was not asked, write `Approved-by:` or a review record, add a label, apply `gate-change-approved`, or release |

A judge answers one question over evidence already collected, read-only, within a small tool budget. The owner opens it as its own session and closes it when the ruling lands; agents do not spawn one, except a coordinator in hold-my-beer mode (#424): while `.agents/review-policy.json` sets `"hold_my_beer": true`, the coordinator starts a judge for proposal and design approval, a design question marked for the owner, a developer's question, a third rejected review, milestone scope or order, a `status:needs-decision` issue, and a `status:blocked` issue whose `Depends on:` issues are all closed. Gate-file approval and releases stay the owner's. A judge's approval is its own `Judged-by:` line, which spec-check fails while the opt-in is off.

### 2. Default route per role

Routes name a tier and band from ADR 0002's tables, never a model, so a new model release changes the policy only.

| Role | Default | Allowed below the default |
| --- | --- | --- |
| `architect` | `frontier` / `high` | No |
| `developer` | `frontier` / `high` | `workhorse` / `high`, only when all four hold (below) |
| `reviewer` | Not weaker than every implementer run (ADR 0002 § 2); a different agent while `cross_provider` is `true` | No |
| `coordinator` | `frontier` / `high` | No: it routes and never implements, so the `workhorse` conditions do not apply |
| `judge` | `frontier` / `high`, one short session | No |

A developer run may take `workhorse` only when:

1. **The target state is already written:** an approved `design.md` with `file:line` premises, or failing tests.
2. **The blast radius is one package** with no exported-surface change.
3. **The failure mode is loud:** tests, lint or schema drift would catch a mistake. Concurrency, ack/durability and readiness fail silently and stay `frontier`.
4. **A reviewer not weaker follows,** which the live gate already requires.

### 3. Escalation and record

- When a developer is escalated mid-change, `implemented-by:` lists every run as comma-separated pairs, for example `claude (sonnet, medium, opus, high)` (ADR 0002 § 3).
- When a role runs below its default, the PR body says so and why.
- The per-model finding rate in `reviewed-by:` records is the long-run check on these defaults. A default changes by amending this ADR.

### 4. Where it is read

- `docs/model-review.md`'s status block links this ADR.
- `.agents/README.md` § Configured model review points here for roles and default routes.
- The protocol's Clarify gate asks which role drafts the deliverable.
- `.agents/contracts/architect.md` points here instead of "pending ADR 0003".

## Not decided here

- **An `architect` or `judge` route in the policy schema or `ateam review route`.** That is `internal/review/`, a gate file (#140). File it once this ADR is accepted and the design track has been used at least once.
- **Mechanical enforcement.** A route is intent; runtime model identity stays an attestation (ADR 0001). Only the reviewer row is enforced, by the live gate.
- ~~**A `judge` contract file.**~~ Resolved by #424: `.agents/contracts/judge.md`, which `ateam init` writes.
- **Adopter templates.** `ateam init` ships the architect, developer, reviewer, coordinator and judge contracts but not this ADR; the defaults here are this repository's.

## Consequences

**Positive.** Every role an agent may be asked to play has one row naming its contract, its write scope and its default tier, inside the repository. The `workhorse` exception is written as four checkable conditions instead of judgment.

**Negative.** Nothing enforces the architect, developer or judge defaults; a run below them is visible only if the PR body says so.

**Neutral.** No policy, gate, ranking or required check changes.
