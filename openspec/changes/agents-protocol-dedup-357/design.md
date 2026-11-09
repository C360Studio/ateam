# Design: `agents-protocol-dedup-357`

## Context

Inventory at `39917f4`, opened this session (architect pass, spot-checked):

- `AGENTS.md` is 15,027 bytes with 25 rule rows (8,311 bytes). Its largest sections are § 3 (5,518) and Four inline workflow gates (2,185). The largest rows are L89 merge:check (958), L90 gate files (798), L85 track (591), L84 proposal (519), L82 team settings (495), L72 hook configs (446) and L91 archive/close (421).
- `.agents/protocol.md` is 24,627 bytes, and `.agents/rules-index.md` 10,010. Neither is imported into a session.
- Code that reads these files:
  - `internal/bootstrap/bootstrap.go:309-352`: the size cap and row-count/section parity only, never row text.
  - `bootstrap.go:279-286`: `protocolMarkers` (protocol only).
  - `internal/bootstrap/settings.go:25-42`: merge-method docs (protocol only).
  - `internal/bootstrap/reviewrecord.go:81-96`: rewrites the adopter rows `agentsMergeCheckRow` and `closeRuleRow` (`templates.go:632,638`). `settings_test.go:141` asserts their merge-method text.
- The adopter `defaultAgentsMD` (`templates.go:175-232`) has no Four-inline-gates section.
- Detail found only in AGENTS:
  - the merge-check PR-classification globs (L89);
  - the team-settings rule (L82; its long form is in `docs/adoption.md` § Team settings);
  - the Close stop rule "never remove a worktree you do not own … dirty or unpushed" (L101).
- Must stay verbatim in AGENTS, because a session may never open the protocol:
  - the Clarify 4-tuple and Stop boundary, and "stop at proposal approval and hand off";
  - worktree and branch naming;
  - "only the owner applies `gate-change-approved`; a blocked agent hands off";
  - "never remove a worktree you do not own, or one that is dirty or unpushed";
  - "never force over a handoff or clear a lease";
  - non-destructive mutation;
  - "the reviewer is not weaker";
  - gate-weakening settings stay out of `ateam.json`;
  - the Ponytail "Never cut" list and footer.

## Goals / Non-Goals

Goals: cut the always-loaded bytes, give each obligation one canonical home, and lose no per-turn rule. Non-goals: protocol rewrites, rule changes, new checks.

## Decisions

### D1. What shrinks

- **A (recommended).** Shrink L89, L90, L84, L82, L72 and L91 to one sentence each (about 200 bytes), keeping the must-stay clauses and adding a pointer. Estimated saving: 3,637 → about 1,200 bytes, so **about 2.4 KB (16%)**. Rows and sections are unchanged, so parity is safe. Forecloses: AGENTS as a standalone merge-check reference; the globs then live in protocol § 4 and `scripts/merge-check.sh`.
- **B.** A, plus L85 (the track trigger list). About 0.4 KB more, but the trigger list decides the Clarify outcome for every non-trivial task, so it is per-turn knowledge. Not recommended.
- **C.** Leave AGENTS alone. 0 bytes saved; 15 KB is well under the cap.

### D2. Four inline workflow gates

- **A (recommended).** Keep the section. Move only the Close step's mechanics (archive path, `gh` flags, the Gitea payload) behind a § 2 step 5 pointer, and keep its stop rule. Saves about 0.6 KB more. The protocol preamble's promise that AGENTS inlines the gates (`protocol.md:3`) still holds.
- **B.** Collapse every gate to one line. Saves about 1.5 KB, but AGENTS stops being a self-contained gate checklist for hosts that never open the protocol.

### D3. Adopter template

- **A (recommended).** Leave `defaultAgentsMD`, `agentsMergeCheckRow` and `closeRuleRow` as they are. The AIP swap (`reviewrecord.go:90-91`) and `settings_test.go:141` keep working. Adopters keep longer rows, but nothing breaks. The repo and the template already differ (the template has no gates section).
- **B.** Shrink the template rows too. That changes adopter surface and needs the swap and test updated together.
- **Adopter seam:** under A, an adopter sees no change.

## Risks / Trade-offs

- A shrunk row loses a clause that a session needed on every turn. Mitigation: the trace in task 3.1, re-traced by the reviewer.
- Byte savings are a proxy, not a measured token or productivity gain.

## Migration Plan

One docs PR: protocol additions first, then the AGENTS shrinks. No state migration.

## Open Questions

- **OQ1 (owner).** D1 A, keeping the track list (L85) in AGENTS?
- **OQ2 (owner).** D2 A, keeping the gates section and moving only the Close mechanics?
- **OQ3 (owner).** D3 A, leaving the adopter template alone?
- **OQ4 (owner).** Team settings (L82): point to `docs/adoption.md` § Team settings (recommended), or add a protocol § 1 paragraph?
