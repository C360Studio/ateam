# Design: queue-checks-416

## Context

- `internal/state/forge.go:898-917`: `classifyCardColumn` puts an issue in the blocked or needs-decision column from its label alone; nothing checks the label.
- `internal/state/next.go:82` (`parseDependsOn`) and `:116-215` (`computeNext`) already parse `Depends on:` from body and comments, treat "not in the open set" as closed, and compute `Unplaced` (`:168-170`): no milestone, or missing from the `Order:` line. `CommentsUnread` marks issues whose comments were not fully read.
- `forge.go:865-896`: `BoardCard.Decision` is set from a comment, body line or label starting `Decision:` or `Ruling:` (`findDecisionLine`). This repository's owner rulings read `Owner decision 2026-11-26:` and `Owner rulings …`, which do not match.
- `forge.go:861`: `BoardCard.UpdatedAt` is the sync time, not the forge's; no issue creation or update date is cached, so "unmilestoned for N days" cannot be computed today.
- Milestones are known only through the cards that carry them (`forge.go:814-818`). A milestone with no open issue has no open card, and closed cards are capped at 50 (`forge.go:452`), so "open milestone with all issues closed" is not computable from the cache.
- `internal/state/state.go:3057-3090`: the `[ateam]` line already carries `since-review=` and `next=` as cache-only advice. `cmd/ateam/main.go:624`: `runDoctor` is offline (parity, MCP budget, hook timing) and runs in `task verify`.
- `internal/state/settings.go:29-38`: `.agents/ateam.json` has no queue keys. A threshold that cannot weaken a merge gate belongs there (settings rule).

## Goals / Non-Goals

**Goals:** make label and milestone drift visible at every session start; one command a person or host workflow runs to list it.
Owner rulings 2026-11-26 (#416 comment): D1 A, D3 A, D4 A (widening filed as #421), Q5 default 15.

**Non-Goals:** fixing labels, gating anything, scheduling, judge placement (#417).

## Decisions

### D1. Blocker grammar

- **A (owner ruling): `Depends on: #n` only**, the form #418 already parses. A `status:blocked` issue that names its blocker another way (`Blocked by #n`, prose) is reported as Q2, "names no readable blocker", which tells the author to use the one form.
- **B: also accept `Blocked by #n`.** Friendlier, but two grammars for one fact, and `next` (#418) would need the second form too or the two answers diverge.
- *Adopter seam:* an adopter using `Blocked by` sees Q2 findings until they switch. Doing nothing leaves them as advice.

### D2. The rules

- **Q1** `status:blocked`, with `Depends on:` issues and none of them open: "blocker closed".
- **Q2** `status:blocked` with no `Depends on:` line: "names no readable blocker".
- **Q3** `status:needs-decision` with `BoardCard.Decision` set: "decision recorded".
- **Q4** unplaced (reuses `NextResult.Unplaced`): no age threshold. The owner's note (#418 comment, 2026-11-26) says issues arrive mid-run and a strict rule would stall. Reporting every unplaced issue keeps them visible, and placing them stays a person's or a judge's call (#417).
- **Q5** an open milestone with more than `queue.max_milestone_issues` open issues (default 15, the adopter's overloaded size).
- **Q6** an open milestone with zero open issues: "finished, not closed".
- An issue with `CommentsUnread` skips Q1 and Q2 (its dependencies are unknown) and is counted under `queue=` as "unread", never as clean.

### D3. Where milestone data comes from (Q5, Q6)

- **A (owner ruling): one open-milestones call per sync** (`gh api repos/{o}/{r}/milestones?state=open`; Gitea `GET /repos/{o}/{r}/milestones?state=open`). It returns `open_issues` and `closed_issues` per milestone, so Q5 and Q6 do not depend on the 50-item closed cap. It costs one request per sync, and the hook stays cache-only.
- **B: count from cards only.** No new call, but Q6 is impossible (a finished milestone has no open card), and Q5 already works from open cards.
- **C: drop Q5 and Q6.** Smallest, but loses the adopter's overloaded-milestone finding.
- *Adopter seam:* a token that can read issues can read milestones on both forges. A failed call leaves Q5 and Q6 out and shows `queue=unavailable`, never zero.

### D4. Q3's decision grammar

- **A (owner ruling): keep `Decision:` / `Ruling:` as is.** Q3 then misses rulings phrased `Owner decision …`, a false negative that is quiet but harmless (advice only).
- **B: also match `Owner decision` and `Owner ruling(s)` prefixes.** Catches this repository's own phrasing, but changes the board's Decision column for every adopter, which needs its own Requirement change.
- B is filed as #421.

### D5. Surfaces

- Pickup line: ` | queue=<n>` after `next=`, or `queue=unavailable` when the milestones read failed or nothing is synced.
- `ateam state queue-check`: one line per finding (`Q1 #12 blocked, but its blocker #5 is closed`). It exits 0 on findings and 1 only without synced data, so a host cron job can tell the two apart.
- `ateam doctor`: one advice line from the cache. `--check` never fails on findings, and doctor makes no forge call, because it runs in `task verify` on CI.
- The preflight skill names `ateam state queue-check` before a handoff. That is the owner's "preflight and/or doctor" ask without a gate.

## Risks / Trade-offs

- Advice can be ignored, and a count nobody reads is the same drift #416 measured. #417's coordinator is meant to read it.
- The Q5 default of 15 is one adopter's datum; the setting exists so a team can move it.

## Migration Plan

Additive. On this repository today the expected findings are Q4 for none (#347 was placed on 2026-11-26) and Q6 for any open milestone without issues. A repository with no milestones shows `queue=0`.

## Open Questions

None. The owner ruled D1, D3 and D4 A and the Q5 default of 15 on 2026-11-26; D2 and D5 follow the recommendation on approval.
