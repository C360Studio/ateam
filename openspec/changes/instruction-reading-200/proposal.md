# Change Proposal: `instruction-reading-200`

Approved-by:

## Why

Every workflow in this repository begins with an unconditional full read of the protocol. The pickup, handoff, preflight, design and review-pr skills each open with "Read the shared protocol first", and all three role contracts say "Read AGENTS.md and .agents/protocol.md fully" (#200). At `3812615`, `AGENTS.md` is 14,881 bytes and `.agents/protocol.md` is 23,971. A pickup therefore reads about 41.7 KB before it does anything, and a review reads about 47.9 KB. The contracts also ask a run to re-read `AGENTS.md`, which Claude, Codex and Gemini have already loaded natively. `ateam doctor`'s <300-token schema figure counts none of this.

## What Changes

- Each canonical skill and role contract names the protocol sections (by `§` number) its workflow needs, in place of the full-read line. Its adopter template in `internal/bootstrap` changes the same way.
- The contracts stop asking for a second read of `AGENTS.md`. They say the host loads it, and to read it only if it is not in context.
- A new harness Requirement, plus a bootstrap test: every `§` a shipped skill or contract names must exist in the shipped protocol template, so a renamed section cannot leave a dangling pointer.
- `tasks.md` records reproducible before and after bytes per workflow (pickup, small implementation, review, handoff, design). Static instructions, hook injection, tool schema and tool responses are reported separately. It also records a manual trace of each workflow to every obligation it needs.

## Track

`design`: this changes adopter templates shipped by `ateam init` (exported surface reachable from adopting repositories) and adds a spec Requirement. See `design.md`.

## Non-goals

- Rewriting the protocol, the review policy (#179), or mode resolution (#91/#92).
- Board responses (#199, shipped) and the MCP schema.
- A new gate, report command or reporting framework. Any claim about agent productivity.
- Trimming the duplicated obligations between `AGENTS.md` and the protocol, unless the owner rules otherwise on design OQ1.

## Definition of Done

- No canonical skill, contract or adopter template asks for a full protocol read. Each names its sections, and the trace in `tasks.md` shows that every obligation of its workflow lives in `AGENTS.md`, the named sections, or the skill itself.
- Before/after bytes per workflow are recorded in `tasks.md` as `wc -c` commands at fixed SHAs, with the four cost classes kept apart.
- The new bootstrap test fails when a named `§` is missing from the protocol template (`task mutate:check` reports `detection`).
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green. The change is archived before the PR is marked ready.
