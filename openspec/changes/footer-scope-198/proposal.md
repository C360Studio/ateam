# Change Proposal: `footer-scope-198`

Approved-by:

## Why

`AGENTS.md:13` tells agents to "end every reply" with 1–2 lines on what was skipped and the residual risk. The Codex dogfood pass at `2d7eeef` showed the result: acknowledgements carry a risk footer before any work has happened (#198). The ritual line hides the real disclosures. The rule exists to make conclusions and handoffs honest, not to tag every message.

## What Changes

One rule, one wording, applied everywhere it is stated:

> End each task conclusion and each handoff with 1–2 lines: what you skipped or did not check, and any residual risk. Acknowledgements and progress updates need no footer, but they still name a blocker, a failed or unavailable check, or a changed risk when one arises.

Sites (all checked in this session):

- `AGENTS.md:13` and its rule row `AGENTS.md:64`; `.agents/rules-index.md:15`.
- `.agents/protocol.md:43` (Gate 4, Ponytail Exit Summary).
- `.agents/skills/ponytail/SKILL.md:5,11`; `.agents/skills/preflight/SKILL.md:31` and `.agents/skills/review-pr/SKILL.md:35`, which already apply to a conclusion and only change "reply" to "conclusion".
- The `ateam init` mirrors in `internal/bootstrap/templates.go:183,204,245,436,442,474,510` and `internal/bootstrap/reviewrecord.go:47`, kept byte-identical to the canonical files.
- The injected ladder in `internal/hook/hook.go:430` (`LadderPrompt`): "End reply with 1-2 lines" becomes "End conclusions and handoffs with 1-2 lines". It stays about the same length. The hook does no extra work, so latency is unchanged.

## Track

`quick`. No spec Requirement is added or changed: the harness spec says only that the ladder is injected, not what its last line says. No exported surface, boot path, durability behavior, or gate file is touched. The generated `init` text changes wording only.

## Non-goals

- Session or turn mode detection, or any runtime that decides which message is a conclusion (#92, v0.4.0). #92 must keep this scope when it lands.
- Tests that grade prose style.
- Frozen benchmark instruction bundles under `benchmarks/`; they are versioned inputs and stay as they are.
- A new footer line in the handoff skill. The `AGENTS.md` sentence already covers handoffs.

## Definition of Done

- Every site above carries the new scope; `grep -rn "end every reply\|End reply with"` outside `benchmarks/` and `openspec/changes/archive/` finds nothing.
- Canonical files and their `templates.go` mirrors match byte for byte (the existing bootstrap parity tests).
- `task verify` is green on a clean tree, and `ateam doctor --check` passes, including the `AGENTS.md`/rules-index row parity and the hook latency budget.
- Direct read of the representative instructions: the ack/progress, conclusion, and handoff wording in `AGENTS.md`, `protocol.md` Gate 4, and the hook prompt.
