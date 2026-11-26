# Design: coordinator-417

Drafted by claude (opus, medium) in the architect role. Read-only on the deliverable; the owner rules.

## Context

- `docs/adr/0003-roles-and-routes.md:24-30`: the catalog has four roles (architect, developer, reviewer, judge) and no coordinator. `:30`: "agents do not spawn one" (a judge). The second change for #417 amends that line; this change does not.
- `internal/bootstrap/bootstrap.go:30`: `canonicalSkills` lists six skills, `design` among them. `internal/bootstrap/templates.go:607-611,625-633`: `ateam init` writes three contracts and, for each canonical skill, the canonical file plus a Claude and a Gemini adapter. Init has no optional-file mechanism: every template is always written.
- `openspec/specs/harness/spec.md:606`: *Adopter skills and adapter aliases* still says "the five skills" and leaves out `design`, which the code already counts. This change corrects that count.
- `.agents/protocol.md:96-104`: unattended and fan-out runs. Each run gets its own worktree, handoffs are keyed `agent@branch` (`internal/state/state.go:693`), and a gate-file change hands off.
- `.agents/hosts/{enjoy,herdr,t3code}.toml` `[launch]`: each host starts a session with a given agent, model, effort and prompt (Enjoy `start_conversation`, herdr `agent start` plus `agent prompt`, T3 `t3_thread_launch`). The manifest text is reviewer-specific, and the manifests are gate files. This answers #417's "not checked": all three hosts can start a session, but no manifest has a developer launch line.
- `ateam state next`, `ateam state queue-check` (#418, #416) and `ateam review request`/`notify` already exist. The coordinator needs no new command.
- `#411` (escalate on a third REQUEST CHANGES) is open. In default mode that escalation is an owner gate.

## Goals / Non-Goals

**Goals:** a written role that moves milestone work from `next` to merge with no owner prompt it could have avoided, batches the prompts it cannot avoid, and counts them.
**Non-Goals:** judges and `Judged-by:` (the second change), any gate-file edit, a scheduler, a new runtime, and label or milestone writes.

## Decisions

Owner rulings already given (#417, 2026-11-26): default mode stops at every owner gate; #417 is split into two changes; human-only gates (release, `gate-change-approved`) never move to an agent.

### D1. Shipping "optional"

- **A (recommended): always written by `ateam init`, inert until invoked.** One contract and one canonical skill, with no new init flag. "Optional" means a team chooses to run a coordinator, not that it chooses to install the files. Cost: two more files in every adopting repository, and doctor applies the canonical-skill checks to them.
- **B: an `ateam init --with coordinator` flag.** Adopters get fewer files, but this adds an option surface, a second init path to test, and a doctor rule for "present or absent".
- *Adopter seam:* if they do nothing, the files sit unused. A team with its own orchestrator ignores them, just as it ignores `design`.

### D2. Handoff slot

- **A (recommended): its own worktree and branch, `.worktrees/<agent>-coordinator` on `<agent>/coordinator`, with no PR.** The slot becomes `<agent>@<agent>/coordinator`, so it cannot collide with `<agent>@main`, and it needs no code. Pickup from `main` already lists each worktree's handoff (protocol.md:102). Cost: a worktree that never merges. Prune skips it because it has no merged PR.
- **B: a role field in the slot key (`claude:coordinator@main`).** No worktree, but it touches `HandoffSlotKey` and every caller, plus a spec Requirement on handoff slots.
- *Adopter seam:* a coordinator started on `main` collides with developer handoffs there. The skill's first step moves it.

### D3. What the coordinator writes

- **A (recommended): handoff, PR comments (relayed review results and batched owner questions), and one `owner-prompts: <n>` line in the PR body.** Archiving, readying the PR, arming auto-merge and cleanup stay with the developer run, which owns the claim (AGENTS.md Close). When that run has ended, the coordinator starts a new one on the same worktree rather than doing the step itself.
- **B: the coordinator also arms auto-merge and cleans up.** One fewer launch per PR, but it then acts on a worktree it does not own, which protocol § 2 step 5 forbids.
- Queue-check findings are reported, never fixed, in default mode (owner ruling: it never changes milestone scope, and label edits such as clearing `status:blocked` are rulings).

### D4. Starting a developer run

- **A (recommended): the host's own launch tool, named in the skill, with a developer prompt.** The skill quotes the tool each `[launch]` line uses (D-Context) and the prompt shape (contract, issue, worktree branch, `implemented-by:`). No manifest change. When the host has no tool, or the coordinator is not in a known host (no `ATEAM_HOST`), it puts the prompt in the batched owner message, and that counts toward `owner-prompts`.
- **B: a `[launch.developer]` table in each host manifest, driven by a new `ateam run request`.** Mechanical and testable, but it edits three gate files and adds a command. Defer this until the baseline shows that hand-off launches are a large share of owner prompts.

### D5. Counting owner prompts

- **A (recommended): a PR-body line `owner-prompts: <n>`.** One batched message that covers k PRs adds 1 to each of them. A launch the owner had to start counts too. Query: `gh pr list --state merged --search "owner-prompts in:body"`. The gate reads only body lines that start with `implemented-by:` or `reviewed-by:` (`internal/review/gate.go:345-349,472-480`), so it ignores this one.
- **B: a run-summary comment on the milestone's last issue.** This gives one number per run instead of per PR, but it divides badly by PR, and runs that end mid-milestone leave no anchor.
- *Adopter seam:* the line is informational. No gate reads it.

## Risks / Trade-offs

- A coordinator and a developer both on Claude under one forge login are indistinguishable to the gate (ADR 0001). The coordinator never writes `reviewed-by:` and never records a review.
- The baseline is measured only on this repository, through Enjoy. herdr and T3 launch paths are written into the skill but not run.

## Open Questions

None that change specs or tasks. Owner rulings are needed on D1–D5 (recommended: A for all).
