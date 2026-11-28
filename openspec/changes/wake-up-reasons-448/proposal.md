# Proposal: wake-up-reasons-448
Approved-by:

## Track

`quick`: it adds one scenario to an existing harness Requirement and no Requirement, touches one capability, adds no exported surface, and is no boot path, durability or gate file. The edits are Markdown instructions and their Go template copies: `.agents/skills/coordinate/SKILL.md` steps 2 and 4 (`internal/bootstrap/templates.go:413`, `:421`) and `.agents/contracts/developer.md` (`internal/bootstrap/review.go:17`).

## Decisions you are making by approving

- **A developer's wake-up line gains three reasons: `claimed`, `holding` and `closed`.** `claimed` is sent once, when the draft PR is open, so the coordinator can start the next unit without waiting for the first handoff. `holding` means the developer waits on the coordinator (for example, told to wait before arming auto-merge) and asks nothing of you. `closed` means merged and cleaned up, so it no longer reads as a review request. `ready`, `blocked` and `question` keep their meaning.
- **The coordinator's launch prompt changes to name all six reasons**, and sends the line "after every handoff, and once your draft PR is open". A developer started from the old prompt keeps sending only the old three, which still work.
- **On the design track, the developer's open questions travel as `question`, naming the `design.md` § Open Questions section instead of a PR comment.** The coordinator reads the question from whichever its handoff names. A proposal awaiting approval with no open question stays `blocked`.
- **A wake-up whose handoff the coordinator has already read is acknowledged and nothing else.** A late or duplicate wake-up starts no review, message or owner prompt.

## Why

#448, from the v0.4.7 coordinator dogfood: the next unit waited for the first handoff because no signal marked the draft PR (note 1); #402's design questions arrived as `blocked` (note 3); #288 and #421 reported their close as `ready` (note 23); a developer told to wait had no word for it (note 40); and wake-ups arrived late or twice (notes 14, 20).

## What Changes

1. Coordinate step 2 (and its template): the launch prompt's reason list and send rule; what the coordinator does on `claimed`, `holding`, `closed`; the duplicate rule.
2. Coordinate step 4 (and its template): the question comes from the PR comment or `design.md` section the handoff names.
3. Developer contract (and `developerContract`): design-track questions and the `holding` reason.
4. Harness spec: one scenario under *Optional coordinator role*, with a shipped-text test.

## Scope & Non-Goals

- **In-bounds:** the two Markdown files above, their copies in `internal/bootstrap/templates.go` and `internal/bootstrap/review.go`, `internal/bootstrap/coordinator_test.go`, the harness spec delta.
- **Non-goals:** any code that parses or checks the reason word (the line is prose a host relays); the coordinator contract's owner-gate list, which a proposal awaiting approval already covers; the judge's wake-up line; the protocol and handoff skill, which a developer without a coordinator follows unchanged. The quoted-command test (#429) sees no new `ateam` command in the prompt.

## Definition of Done

- A test asserts the new sentences in the shipped developer contract and coordinate skill, shown able to fail with a non-destructive mutation of each template; parity keeps the repo files equal to the templates.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
