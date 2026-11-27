# Design: third-rejection-stop-411

## Context

- `.agents/contracts/coordinator.md:7`: the coordinator stops a unit at "a third `Verdict: REQUEST CHANGES` on one pull request"; `:9` sends it to a judge in hold-my-beer mode. `.agents/skills/coordinate/SKILL.md:22`, `:30` repeat both. The count is the coordinator's own reading; nothing in `ateam` counts.
- `.agents/contracts/judge.md:9`: after a third rejection the judge rules per finding whether it blocks, never writes a record; "a fresh reviewer still re-reviews". Nothing records that a round 4 was authorised.
- `.agents/contracts/coordinator.md:5`: the coordinator writes only its handoff, relay and owner-question comments and `owner-prompts:`; it never changes labels.
- `.agents/protocol.md:15`: `status:needs-decision` is listed as an issue label. `internal/state/queuecheck.go:237` flags an issue carrying it with a decision recorded.
- `internal/state/forge.go:277-285`, `:308-315`: sync reads every comment of an open PR with a local worktree and keeps only the newest record (`latestReview`). `internal/state/state.go:597-603` caches it as `Review`.
- `internal/state/forge.go:757` (GitHub, `gh pr view --json comments`) and `:412` (Gitea, `fetchGiteaIssueComments`, one page, `:553-576`): whether either returns every comment on a long PR is unverified; task 2.1 checks it first.
- `cmd/ateam/reviewlaunch.go:151-261`: `ateam review request`; `:214-227` sends a round back to `--reviewer-session` and returns before the fresh-launch path; `:321-332` `cachedRecord` already reads the sync cache. Exit 2 already means "refused, nothing run" (spec `Review request spec`, scenario *Reviewer the gate would reject*).
- `internal/review/` is a gate file (protocol § 4); `ReadPullBody` (`internal/review/gate.go:188`) reads body and head only, no comments.
- `internal/bootstrap/templates.go:370`, `:372`, `:386`, `:415`, `:423` carry the coordinator, judge and coordinate text `ateam init` ships; `internal/bootstrap/coordinator_test.go:45` pins the third-rejection phrase.
- PR #437 (#412) rewrites protocol § 2 step 5 and leaves the coordinator contract and skill alone; this change does not touch § 2 step 5.

## Goals / Non-Goals

- Goal: a mechanical stop at the third rejecting record that a coordinator or a hand-run developer cannot step past by accident, and a recorded owner (or judge) decision that lifts it for one round.
- Non-goals: changing the merge gate; machine-readable severity; `ateam` writing comments or labels; a forge workflow.

## Decisions

### D1. Where the count runs

- **A (recommended): sync caches the rejecting records; `ateam review request` refuses.** Every review launch and same-session re-review already passes through `request`, and the coordinator syncs before reading a verdict (`SKILL.md:22`). Cost: one cached slice and one check in `cmd/ateam`; no gate file. Forecloses: nothing; a workflow can be added later. Staleness: a request before the sync that read the third record launches once more; the coordinator's prose stop stays as the backstop.
- **B: `request` reads comments live.** Fresh, but needs an exported comment reader in `internal/review/` (gate file: owner label on every push) or a second reader in `cmd/ateam`. Rejected on cost.
- **C: sync posts a summary comment and label.** A new write side effect on every sync, idempotency to manage, and the coordinator's runs would then change labels. Does not stop the next launch.
- **D: prose only (status quo).** Free; no stop outside `/coordinate`.
- Adopter seam: an adopter on `review_launch: manual` who runs `request` by hand meets the refusal at round 4; one who never runs `request` sees nothing new.

### D2. What lifts the stop (owner ruling: confirm A)

- **A (recommended): one line per extra round.** An unfenced PR comment line `review-rounds-continue: <n>`, where `<n>` is the rejecting count it answers. `request` passes when count < 3 or the newest such line names exactly the current count. A fourth rejection needs `review-rounds-continue: 4`. Mirrors `gate-change-approved: <sha>`: bound to the state it answers, so it cannot be reused.
- **B: one line lifts all later rounds.** Fewer prompts; but semsource#365's fourth round would have passed on the third round's decision.
- **C: a `request` flag (`--owner-continue`).** Leaves no record on the PR, and the run that wants the round would set it.
- Who writes it: only the owner, or the judge in its ruling comment under hold-my-beer (`judge.md:9`). Developers and the coordinator never do. Under a shared forge login this is an attestation, as `gate-change-approved` is.
- Adopter seam: the stderr text names the exact line to post.

### D3. Summary content (owner ruling needed)

- **A (recommended): facts only.** Per rejecting round: reviewed sha, reviewer run, record link; plus the first reviewed sha and the PR head. Severity and reachability are judgment the judge already makes (`judge.md:9`) or the owner reads from the links.
- **B: severity in the record grammar.** A `Blocking: <n>` line in reviewer records (`reviewer.md` Record, review-pr skill, templates, parser). The issue asks for it; it changes what every reviewer writes and still counts, not weighs.
- **C: count unfenced `Non-blocking:` lines.** No grammar change, but the blocking side has no marker, so the number misleads.
- Forecloses: A leaves room for B later; the summary only gains a column.

### D4. Labelling (owner ruling needed)

- **A (recommended): no label.** The refusal stops the launch; the coordinator already stops the unit and gathers the question (`SKILL.md` step 4). Keeps `status:*` an issue label and the coordinator's "never change labels".
- **B: `request` labels the PR `status:needs-decision`.** Visible on the board, but a command the coordinator runs would change labels, contradicting `coordinator.md:5`; and someone must remove it.
- **C: label the issue.** `ateam state next` then skips it; `queue-check` flags it once a decision is recorded (`queuecheck.go:237`), so it needs a removal step too.

## Risks / Trade-offs

- A Gitea PR with more comments than one page could undercount until task 2.1's check; if so, use `fetchGiteaAllIssueComments` (`forge.go:585`) for PRs too.
- A stale cache lets one extra round through (D1 A); accepted, the coordinator's stop remains.
- Exit 2 now has a third cause; the stderr reason distinguishes it.

## Migration Plan

None: the cache field is additive; an old cache has no rejecting list and counts 0 until the next sync.

## Open Questions

- D2 (confirm A), D3, D4: owner rulings, listed in `proposal.md`.
