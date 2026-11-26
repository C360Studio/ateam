# Proposal: hold-my-beer-424
Approved-by: Coby Leuschke (2026-11-26) sha256:9123ed3384d7

## Track

`design`: adds a harness Requirement, touches gate files (`internal/review/`), adds a judge contract that `ateam init` ships, and amends ADR 0003. This is part 2 of #417; part 1 (PR #423) shipped the default-mode coordinator.

## Decisions you are making by approving

- A repository can switch on a mode where the coordinator asks a judge instead of you at some of your decisions. It is off unless the review policy turns it on, and turning it on is a gate-file change that needs your label.
- With the mode on, a judge may approve a proposal and its design, answer a design question marked for you, rule after a third rejected review, and set milestone scope and order, including placing issues that arrive during a run. Releases and gate-file approvals stay yours, always.
- A judge's approval is written on its own line, never on your approval line, and names the judge's agent, model and effort. A comment beside it carries the question and the evidence the judge read. If you also approve, your approval is the one that counts.
- With the mode off, a judge's approval line fails the spec check, so it cannot stand in for you by accident.
- The judge must be at least as strong as every run it rules on, and from a different provider than the author when the policy asks for that. The coordinator checks this before it starts the judge; nothing checks it afterwards.
- A judge never turns a rejected review into an approval. After a third rejection it rules whether the finding blocks, and a reviewer still has to approve before the pull request merges.
- A judge-run ends and hands back to you when the judge says it cannot decide, `main` goes red, a change touches a gate file, or the milestone it started on is finished. There is no cap on how many pull requests a run merges.
- A judge may edit a milestone's order line and an issue's milestone, and it records each edit as a comment on the issue.

## Why

The owner-prompt count from default mode is the cost hold-my-beer removes. ADR 0003 says agents never spawn a judge, the judge has no contract, and `Approved-by:` is the owner's alone, so there is nowhere for a delegated ruling to go that cannot be mistaken for the owner's.

## What Changes

1. `internal/review/policy.go`: a `hold_my_beer` boolean, off by default.
2. `internal/state`: a `Judged-by:` line. It is left out of the approval digest, honoured only with the opt-in, and can go stale like `Approved-by:`. It appears in `spec-check`, `spec-queue` and the `ateam/proposal` status.
3. `internal/review/gate.go`: the merge-check note names a judged change as judged, not as unapproved.
4. `.agents/contracts/judge.md` (written by `ateam init`), a hold-my-beer section in the `coordinate` skill and the coordinator contract, and the ADR 0003 amendment.
5. `openspec/specs/harness/spec.md`: one ADDED Requirement.

## Scope & Non-Goals

- **In-bounds:** the above, plus `docs/model-review.md` and `docs/daily-use.md`.
- **Non-goals:** turning the mode on in this repository; a `judge` route in the policy's `routes`; a mechanical strength check on the judge; #411's escalation command (v0.4.7); any agent path to `gate-change-approved` or a release.

## Definition of Done

- Table tests cover `Judged-by:` with the opt-in on and off, alongside `Approved-by:`, and stale, unbound and fenced lines. A digest test shows adding either line leaves the digest unchanged. `task mutate:check` detects the removal of the opt-in check.
- After approval, a coordinator session (default mode) moves this change's PR, and the PR body carries `owner-prompts:`, which also counts on #417.
- The owner applies `gate-change-approved` at the head. `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before ready.
