# Design: review-blocking-bar-410

## Context

Read this session at `d8c3c19`:

- `.agents/contracts/reviewer.md:7` — "Inspect the complete base-to-head diff…"; no statement of what blocks.
- `.agents/contracts/reviewer.md:11` — "Approval requires no blocking/high findings and all required verification"; "blocking" is never defined.
- `internal/bootstrap/review.go:34,38` — the same two sentences in the template `ateam init` writes.
- `.agents/skills/review-pr/SKILL.md:35` and `internal/bootstrap/templates.go:535` — "A re-review message names the commit you reviewed and the new head: … review the full base-to-head diff again as well as the commits since yours".
- `cmd/ateam/reviewlaunch.go:388-391` — `rereviewText` tells the reviewer to "review the full base-to-head diff and <record>..<head>"; pinned by `cmd/ateam/reviewlaunch_test.go:453`.
- `openspec/specs/model-review/spec.md:293` — "Re-review to the reviewing session" requires only that the text names the record sha and the head, so it stays valid.
- `internal/bootstrap/bootstrap.go:446` and `parity_test.go:113-133` — the shipped `review-pr` body must equal the repository copy.
- `AGENTS.md` "Never cut" list — security, trust-boundary validation, data-loss error handling, tests on non-trivial logic.

## Goals / Non-Goals

Goals: a reviewer can tell from the contract alone whether a finding blocks; a later round costs a delta read, not a full reread. Non-goals: escalation (#411), archive timing (#412), CLI flags, gate code.

## Decisions

**D1. Where the blocking bar lives.**
- A (chosen): a default bar in the reviewer contract; a repository raises it through its own project rules, which the contract already says to apply. Cost: two copies of one paragraph. Forecloses nothing; a flag can be added later.
- B: `--bar <text>` on `ateam review request`, printed into the launch prompt. Cost: flag, prompt plumbing, a spec scenario. Rejected by the owner on 2026-11-24.
- Bar contents: regression against the base; silent data/ID corruption reachable from a supported input; a spec or contract violation shown by a reproduction or a traced path; a breach of an AGENTS.md never-cut rule; missing or failing required verification. Everything else is listed under `Non-blocking:` and the author files it as a follow-up before merge.
- Adopter seam: adopters who rerun `ateam init` get the new contract; one who does nothing keeps the old contract and the old behaviour.

**D2. Backstop credit.** Each finding states whether an existing check already catches it. A caught finding is non-blocking unless the spec makes the earlier refusal load-bearing and the reviewer says why.
- Alternative: leave it to reviewer judgment. That is how semsource#365 round 4 blocked.

**D3. Scope of a later round.**
- A (chosen): confirm closure of each prior finding, then review `<record>..<head>`; widen to base..head when the delta touches files outside the prior diff, or when asked. Cost: skill paragraph, contract sentence, one `fmt.Sprintf` and its test. Forecloses: a later round no longer rereads unchanged code by default; a defect in unchanged code it would have found in a reread waits for a full review.
- B: keep the full reread and rely on the bar alone. Cheaper to write, but every round still grows with the diff (31 commits on semsource#365).
- Adopter seam: the re-review text comes from the `ateam` binary, so adopters get it on upgrade with no file change.

**D4. Diverging reviewers.** The latest record `task merge:check` reads already gates the merge. The contract adds: another reviewer's conflicting verdict is quoted as dissent in the record, and the deciding reviewer answers each of its findings. No gate change.

## Risks / Trade-offs

- A narrower later round can miss a defect that a full reread would catch. The bar still blocks it when it is found, and the full reread stays available on request.
- "Supported input" and "traced path" are judgment words; the bar narrows argument but does not end it.

## Migration Plan

None: wording and one message string. Open records stay valid.

## Open Questions

- Resolved by the owner on 2026-11-24: a breach of a never-cut rule (D1, item 4) blocks.
