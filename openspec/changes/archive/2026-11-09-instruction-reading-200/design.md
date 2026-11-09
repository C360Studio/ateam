# Design: `instruction-reading-200`

## Context

Measured at `3812615` with `wc -c` and opened this session.

- Always loaded: `AGENTS.md` (14,881 bytes). Claude imports it via `CLAUDE.md` (20), Codex loads it natively, and Gemini loads it through `.gemini/settings.json:3` (`context.fileName`, checked by `internal/bootstrap/bootstrap.go:816-841`).
- Full-read mandates: `.agents/skills/{pickup,handoff,preflight,design,review-pr}/SKILL.md:10` ("Read the shared protocol first"); `.agents/contracts/{architect,developer,reviewer}.md:3` ("Read AGENTS.md and .agents/protocol.md fully"). Adopter mirrors: `internal/bootstrap/templates.go:351,368,397,434,482,501` and `internal/bootstrap/review.go:19,30`.
- `.agents/protocol.md` (23,971) by section: preamble 243; Adoption scope 944; § 1 State 2,492; § 2 Lifecycle 7,524; § 3 Mutation/PBT 2,766; Test-to-spec attribution 3,072; § 4 Review (incl. Unattended) 6,930. The adopter template keeps § 1–4 numbering (`templates.go:280-322`); its § 4 heading differs ("Configured model review"), so pointers use numbers, not titles.
- Static reads per workflow today (AGENTS + protocol + skill/contract): pickup 41,720; handoff 41,647; small implementation (developer + preflight) 42,329; review (reviewer + review-pr) 47,938; design (architect + design) 41,723. A contract's "Read AGENTS.md" can add a second 14,881 when the host already loaded it.
- Other cost classes, which this change only records: the SessionStart hook injection (~2,050 bytes at build `11e8871`), the MCP schema (257 tokens, #199 evidence), and tool responses (board, bounded by #199).
- The protocol is also read by code: `bootstrap.go:247-285` (marker checks) and `internal/bootstrap/settings.go:33` (merge method). Both stay as they are.
- Duplicated obligations: AGENTS "Four inline workflow gates" (2,184) restates § 2 steps 1, 2, 4 and 5 (the Close step at `protocol.md:45` alone is 2,230). Many AGENTS rule rows restate protocol sentences. The protocol preamble (`protocol.md:3`) says AGENTS inlines the gates on purpose.

## Goals / Non-Goals

Goals: remove the unconditional full read; keep every obligation reachable; publish reproducible numbers by cost class.
Non-goals: rewriting protocol content, a new gate or report, productivity claims.

## Decisions

### D1. How a workflow finds its obligations

- **A (recommended). Section pointers.** Each skill or contract names its sections: pickup § 1, § 2; handoff § 1, § 2 step 5; preflight and developer § 2, § 3, Test-to-spec attribution, § 4; review-pr and reviewer the same; design and architect § 2 step 1. Estimated protocol bytes per path: pickup/handoff 10,016 (from 23,971); developer/review 20,292; design 7,524. Cost: one-line edits in 8 files and their 8 template mirrors. Forecloses nothing; the protocol stays one file.
- **B. Split the protocol into per-workflow files.** Smallest reads, but it breaks the `bootstrap.go:247-285` and `settings.go:33` readers, every adopter's links, and the single canonical home. Heavy churn.
- **C. Status quo.** No cost, no saving.
- **Adopter seam:** templates change only for new `ateam init` runs or `--force`. An adopter who does nothing keeps the full-read lines. That is correct but costly, and nothing breaks.

### D2. The contracts' re-read of AGENTS.md

- **A (recommended).** Change it to "AGENTS.md (your host loads it; read it if it is not in context)". This saves up to 14,881 per contract run. It forecloses nothing.
- **B.** Keep it. Safe for a host that does not auto-load AGENTS.md, at the cost of a duplicate read everywhere else.
- **Adopter seam:** same as D1. A host without native AGENTS.md loading is still told to read it.

### D3. Guard against dangling pointers

- **A (recommended).** A bootstrap unit test: every `§ n` named in a shipped skill or contract template resolves to a `## n.` heading in the protocol template. It is a `task test:unit` test, not a new gate or doctor check.
- **B.** No test. Risk: a renumbered section silently strands a pointer.

### D4. Measurement

- **A (recommended).** `wc -c` command lines and their outputs, at fixed SHAs, in `tasks.md`, split by the four cost classes.
- **B.** An `ateam doctor` reading-cost report. Rejected: #200 rules out a new reporting framework.

## Risks / Trade-offs

- An obligation that lives only in an unnamed section gets dropped. Mitigation: the manual trace in task 3.1 maps each workflow's obligations to where they are read, and the reviewer re-traces it.
- Byte counts are a proxy. The proposal claims no token or productivity gain.
- Adopters keep the old templates until `--force`. That is intended.

## Migration Plan

Edit the repo files and templates together in one PR. No data or state migration.

## Open Questions

- **OQ1 (owner ruling).** Should this change also cut duplicated obligations between `AGENTS.md` and the protocol? (a) No: record them here and file a follow-up (recommended, smallest change). (b) Yes: make the protocol canonical and shrink AGENTS rows to one sentence plus a pointer. This shrinks the always-loaded file, but touches `rules-index.md` parity and every adopter's AGENTS template.
- **OQ2 (owner ruling).** Is the "if it is not in context" wording in D2 A acceptable for Codex subagents? I have not checked whether Codex child runs load AGENTS.md.
