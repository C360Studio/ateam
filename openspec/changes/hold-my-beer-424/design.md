# Design: hold-my-beer-424

Drafted by claude (opus, medium) in the architect role. Read-only on the deliverable; the owner rules.

## Context

- `docs/adr/0003-roles-and-routes.md:30`: "agents do not spawn one" (a judge). `:63-67`: neither a judge route nor a judge contract was decided there.
- `internal/state/approval.go:38-56`: `approvalLine` finds the first unfenced `Approved-by:`. `:58-73`: `ApprovalDigest` hashes `proposal.md` without that line, plus `design.md`. `:86-101`: `approvalOf` classifies it as unapproved, approved, stale or unbound.
- `internal/state/openspec.go:688-710`: `CheckOpenSpec` fails a stale approval and warns on an unbound one. It never fails an unapproved change: the `OPEN-Q` hold and the convention that only the owner fills the line are what gate it.
- `internal/state/commitstatus.go:114-129`: `ateam/proposal` stays pending while `ApprovedBy` is empty or `OPEN-Q` holds, and otherwise says "approved by <name>".
- `internal/review/gate.go:409-425`: `approvalNotes` prints "has no Approved-by; the spec was not human-reviewed" and never changes the verdict. Requirement *Merge check notes an unapproved OpenSpec change* (`openspec/specs/harness/spec.md:667`).
- `internal/review/policy.go:13-42`: `Policy` has no judge field. `internal/state/designpaths.go:26-40` already reads `.agents/review-policy.json` from `internal/state` through `review.ParsePolicy`, and spec-check can reuse that read.
- `.agents/contracts/coordinator.md` (PR #423): default mode stops at every owner gate, and the coordinator counts REQUEST CHANGES itself. #411's escalation command is open in v0.4.7.
- Owner rulings 1–5 on #417 (2026-11-26), copied into #424, fix the gate list, the separate record, the opt-in's home, judge strength and the stop conditions. The #418 comment rules that the judge edits the `Order:` line.

## Goals / Non-Goals

**Goals:** a delegated ruling that can never be read as the owner's, is honoured only while the opt-in is on, is digest-bound like an owner approval, and leaves the reviewer bar where it is.
**Non-Goals:** enabling the mode here, a policy `judge` route, mechanical judge-strength checks, and #411.

## Decisions

### D1. The opt-in

- **A (recommended): `"hold_my_beer": true` in `.agents/review-policy.json`**, absent meaning false. It covers ruling 1's whole gate list. Cost: one field and one parse test.
- **B: `"judge_gates": [...]`, a list of gates.** This gives per-gate control nobody has asked for, and more schema to validate.
- *Adopter seam:* off by default, and turning it on is a gate-file change. If you do nothing, nothing changes.

### D2. The record

- **A (recommended): a `Judged-by: <agent> (<model>, <effort>) sha256:<12 hex>` line in `proposal.md`, below `Approved-by:`.** `ApprovalDigest` drops both lines, so the digest the judge binds to is the one the owner would, and filling either line never changes it. A filled `Approved-by:` wins. With the opt-in off, a filled `Judged-by:` fails `spec-check` (`judged approval without hold_my_beer`). With it on, the change counts as approved for the queue and the status (`judged by <agent> (<model>, <effort>)`), and it goes stale on the same digest rule. The judge also posts one PR comment holding the line, the question and the evidence paths. A ruling that is not on a proposal (a design question, an escalation, a milestone edit) is only that comment, on the PR or issue.
- **B: comment only, with no line in `proposal.md`.** No parser change, but spec-check and the queue cannot see it, so a judged change still looks unapproved and its hold has no recorded basis.
- *Adopter seam:* a reader who greps for `Approved-by:` sees only owner approvals. Judged changes show under their own label.

### D3. Judge strength

- **A (recommended): an attestation the coordinator checks before launch.** It runs `ateam review compare --implementer <each run ruled on> --reviewer <judge run>` and quotes the result in the judge's comment. This follows ADR 0001: runtime identity is attested either way.
- **B: a `Drafted-by:` line plus a spec-check comparison.** Mechanical, but it adds a second line format and still checks only self-reported names.
- *Adopter seam:* a judge launched by hand skips the check. The comment then lacks the compare output, and a reviewer can see that it is missing.

### D4. Third REQUEST CHANGES without #411

- **A (recommended): the coordinator counts records (as it does today), and in hold-my-beer mode asks the judge whether each open finding blocks under the reviewer contract's bar.** The developer fixes any finding the judge says blocks. Findings it rules non-blocking become follow-up issues. Either way a fresh reviewer re-reviews, and merge still needs `Verdict: APPROVE`. The judge never writes a review record.
- **B: wait for #411.** Simpler here, but a run then stops at every third rejection, which ruling 1 says the judge may answer.

### D5. Launch, contract and ADR

- **A (recommended): `.agents/contracts/judge.md`, written by `ateam init`.** It holds one question, the evidence the coordinator collected, read-only except the D2 record and the D6 edit, at most 20 tool calls, and "cannot decide" as a valid answer. The coordinator starts it with the host launch tool from #417 D4 at ADR 0003's `frontier`/`high` default. ADR 0003 `:30` reads: "agents do not spawn one, except a coordinator in hold-my-beer mode (#424)". `:67` is resolved.
- **B: also add `routes.judge` to the policy schema.** This is ADR 0003's deferred item; nothing reads it yet.

### D6. Milestone scope and order

- Owner ruling (#418): the judge edits the milestone's `Order:` line, or an issue's milestone, and comments on each issue it moves with the old and new line. `ateam state next` and `queue-check` pick the edit up on the next sync. No option is offered here.

## Risks / Trade-offs

- A judge and an author on one forge login are indistinguishable (ADR 0001). The `Judged-by:` line is self-reported, like `reviewed-by:`.
- Judge rulings on design questions and milestones are PR and issue comments that no check reads. An audit is a forge search for `Judged-by:`.

## Open Questions

None that change specs or tasks. Owner rulings are needed on D1–D5 (recommended: A for all).
