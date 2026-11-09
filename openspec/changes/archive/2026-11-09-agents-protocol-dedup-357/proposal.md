# Change Proposal: `agents-protocol-dedup-357`

Approved-by: Coby Leuschke (2026-11-09)

## Why

`AGENTS.md` (15,027 bytes at `39917f4`) is loaded into every session on every turn. Claude imports it through `CLAUDE.md`, Codex loads it natively, and Gemini loads it through `context.fileName`. Since #200, `.agents/protocol.md` is read per section. Several AGENTS rule rows still restate workflow detail that only one workflow needs: merge-check PR classification, the gate-file ritual, proposal mechanics, team-settings keys, hook file lists, and archive mechanics. The protocol already carries most of that detail, or should. Each such byte is paid on every turn by every session and subagent (#357, deferred from #200 OQ1).

## What Changes

- Six workflow-specific rows in `AGENTS.md` (§ 3 L82, L84, L89, L90, L91; § 2 L72) shrink to one sentence. Each sentence keeps the stop, never and trust-boundary clauses verbatim, and points to `protocol § n` (or `.agents/README.md`, or `docs/adoption.md`) for the detail.
- Detail that exists only in AGENTS moves to its canonical home first:
  - the merge-check PR-classification globs → protocol § 4;
  - "never remove a worktree you do not own, or one that is dirty or unpushed" → protocol § 2 step 5 (AGENTS keeps the clause).
- `.agents/rules-index.md` stays the detailed matrix. Its row count and sections are unchanged, so the parity check (`internal/bootstrap/bootstrap.go:309-352`) holds.
- `tasks.md` records before and after bytes, and a trace showing every obligation each shrunk row carried still lives in AGENTS or in a named section.

## Track

`design`: it rewrites stated repo-wide rules in the always-loaded file, and one open question decides whether the adopter template (`internal/bootstrap/templates.go`) changes too. See `design.md`.

## Non-goals

- The "Four inline workflow gates" section stays (design D2).
- No rule is added, removed or weakened. The row count is unchanged.
- No change to `protocolMarkers`, the 24 KB cap, or `CheckMergeMethodDocs`.
- No productivity claim from byte reduction.

## Definition of Done

- The always-loaded `AGENTS.md` shrinks by about 2.4 KB (an estimate; the measured figure goes in `tasks.md`). Every obligation removed from a row is traced to its retained clause or to a named canonical section.
- `ateam doctor --check` parity and `TestAgentSetupParity` are green. Row counts match `rules-index.md`.
- `task verify` and `ateam bench --selftest` are green. A reviewer that is not weaker re-traces the shrunk rows. The change is archived before the PR is marked ready.
