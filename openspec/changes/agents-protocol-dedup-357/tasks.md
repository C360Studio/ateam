## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1–OQ4 (#357).
  - Hold: OPEN-Q proposal awaits owner approval

## 2. Canonical homes first

- [ ] 2.1 Add the merge-check PR-classification globs to protocol § 4. Add "never remove a worktree you do not own, or one that is dirty or unpushed" to § 2 step 5.
  - Premise: the globs exist only at `AGENTS.md:89`; the stop rule exists only at `AGENTS.md:101`.
  - Acceptance: `grep` finds each in `.agents/protocol.md`, and `scripts/merge-check.sh` classification agrees with the globs.

## 3. Shrink

- [ ] 3.1 Shrink `AGENTS.md` L72, L82, L84, L89, L90 and L91 (D1), and the Close bullet's mechanics (D2), to one sentence each plus a pointer.
  - Acceptance: the row count is unchanged and `ateam doctor --check` is green. A trace table maps every removed clause to its retained AGENTS clause or its named section.
- [ ] 3.2 Record `wc -c AGENTS.md` before (`39917f4`) and after, overall and per row.

## 4. Verify

- [ ] 4.1 `task verify`, `ateam bench --selftest`; archive before ready.
