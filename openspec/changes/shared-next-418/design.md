# Design: shared-next-418

## Context

- `.agents/skills/pickup/SKILL.md:36` ends pickup with "Next authorized step" and names no source. `.agents/skills/handoff/SKILL.md:31` puts it in the Goal, and handoffs are keyed `agent@branch`, so each session reads its own copy.
- `internal/state/state.go:587`: `BoardCard.Milestone` is the title only. `internal/state/forge.go:239-241`: the payload's `Milestone` struct decodes only `title`, and `:782` copies it.
- Measured: `gh issue view 418 --json milestone` returns `number,title,description,dueOn`, so the GitHub issue list already carries descriptions. Gitea's v1 issue `milestone` object carries `description` (API schema; not run against an instance).
- `forge.go:660-665`: the GitHub open-issue list asks for `body,comments` and on error or timeout retries **without comments**. `:400-427`: Gitea fetches each issue's comments into `CommentBodies`, and `:817-834` already reads both for decision lines. Owner dependencies on #416 and #417 were posted as comments, not in the body.
- `forge.go:655`: `prFields` omits `body`, so nothing links an open PR to the issue it claims. Protocol Claim requires `Closes #n` or `Addresses #n` in the draft PR body.
- `forge.go:638`: open items are listed up to 1000, closed ones capped at 50 (`closedItemCap`). An issue absent from the open set is closed.
- `internal/state/state.go:3057-3083`: `BuildPickupSummary` prints the `[ateam]` line from the cache. `internal/hook` reads only the cache. `cmd/ateam/main.go:133-170` dispatches `ateam state` subcommands.
- `forge.go:338-352`: the tea-CLI fallback lists 30 items and no descriptions.

## Goals / Non-Goals

**Goals:** one "next" that every session computes from the same forge data; each skip explained; no guess when the inputs are incomplete.
**Non-Goals:** queue-health findings and doctor/preflight checks (#416), a coordinator or judge placement (#417), a scheduler.

## Decisions

Owner rulings 2026-11-26 (#418 comment): milestones are ordered by `After: <title>`; an issue missing from `Order:` is reported and skipped; a milestone with no `Order:` is reported as "no order"; an unmilestoned issue is never next.

### D1. Where the inputs are read

- **A (recommended): at sync, into the cache.** This adds the milestone description, `DependsOn []int` (body plus comments) and `Claims []int` (PR body keywords) to the cards. `next` and the pickup line then compute from the cache. It costs one `body` field on the PR list and no new forge call, and the hook stays inside its 10 ms budget.
- **B: live forge calls inside `next`.** Fresher, but a second forge read path in two API dialects, and the pickup line could not show it.
- *Adopter seam:* `next` is only as fresh as the last `ateam state sync`, like `behind=`. If you do nothing, an older cache shows `next=none (not synced)`.

### D2. Choosing the earliest milestone

- Only open milestones whose description has an `Order:` line are candidates. A milestone is earliest when its `After:` names no open candidate; an `After:` naming a closed or unknown milestone is satisfied.
- **A (recommended): exactly one earliest, or name nothing.** Two roots report `milestone order ambiguous`, and a loop reports `milestone order cycle`. This matches the owner's "no order means name nothing".
- **B: break ties by lowest milestone number.** Never stalls, but silently picks one, which is the drift #418 removes.
- *Adopter seam:* a repository adopting this adds `After:` to all but one ordered milestone. Today this repository's v0.5.0 and v0.6.0 carry the order only in prose.

### D3. When nothing in the earliest milestone can start

- **A (recommended): stop there.** Print `next=none (all of <milestone> skipped)` and list the reasons. A milestone gates its release, and working ahead hides why it stalled.
- **B: fall through to the next milestone in `After:` order.** Keeps a second session busy, at the cost of work starting out of milestone order with no ruling.
- A milestone with no open issue in its `Order:` line counts as done for D2, so the next one becomes earliest.

### D4. What counts as claimed

- **A (recommended): any open PR, draft or not, whose body has `Closes`, `Fixes`, `Resolves` or `Addresses` followed by `#n`** (case-insensitive, as GitHub reads closing keywords, plus the protocol's `Addresses`).
- **B: also infer from branch or worktree names** (`claude/418-…`). This catches a worktree with no PR yet, but the number in a branch name is convention, not contract.
- *Adopter seam:* a session that claims without a draft PR is invisible to `next`, as it already is to the board.

### D5. Dependencies, and comments that were not read

- `Depends on: #n[, #m]` at the start of a line, in the body or in any comment, case-insensitive. A dependency is open when it is in the open set (`forge.go:638`). `Blocked by` stays with #416.
- **A (recommended): when the sync fell back without comments, name nothing (`dependencies unread`).** A missed dependency would hand out blocked work, a silent failure.
- **B: compute from bodies only and add a warning.** Always answers, but can name an issue that is waiting on another.

### D6. Output

- Pickup line: ` | next=#418` or ` | next=none (<reason>)`, after `holds=`.
- `ateam state next` prints the milestone, the next issue and its title, one line per skipped issue with its reason (`claimed by #PR`, `waits on #n`, `status:blocked`, `status:needs-decision`), and an `unplaced:` list (milestoned issues missing from `Order:`, plus unmilestoned issues). It exits 0 with a result and 1 when sync data is missing. The same fields go into `ateam state status` JSON.
- The owner's note (#418 comment) wants a preflight or doctor finding for unplaced issues. #416 owns that finding. `next` only lists them, and judge placement belongs to #417.

## Risks / Trade-offs

- Grammar is a contract: a typo (`Order #1`) reads as "no order", which is loud, not silent.
- The 50-item closed cap does not matter: a dependency counts as open only when it is in the open set.
- Gitea descriptions come from the API schema and are unverified on a live instance until task 2.1's fixture runs against the test credentials.

## Migration Plan

Additive. Without `Order:` lines, pickup shows `next=none (no ordered milestone)`. Task 2.5 adds `After:` lines to this repository's milestones once the owner confirms them.

## Open Questions

1. **Owner ruling:** D2 tie: name nothing (A), or lowest milestone number (B)?
2. **Owner ruling:** D3: stop at the earliest milestone (A), or fall through (B)?
3. **Owner ruling:** D5: comments unread: name nothing (A), or bodies plus a warning (B)?
