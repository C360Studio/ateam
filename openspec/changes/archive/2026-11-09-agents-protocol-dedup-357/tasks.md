## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1–OQ4 (#357).
  - Evidence: the owner approved in Enjoy on 2026-11-09, with every question ruled as recommended. OQ1: D1 A, and the track list (L85) stays. OQ2: D2 A, the gates section stays and only the Close mechanics move. OQ3: D3 A, the adopter template is untouched. OQ4: team settings point to `docs/adoption.md` § Team settings. Claude wrote the header at their request.

## 2. Canonical homes first

- [x] 2.1 Add the merge-check PR-classification globs to protocol § 4. Add "never remove a worktree you do not own, or one that is dirty or unpushed" to § 2 step 5.
  - Premise: the globs exist only at `AGENTS.md:89`; the stop rule exists only at `AGENTS.md:101`.
  - Acceptance: `grep` finds each in `.agents/protocol.md`, and `scripts/merge-check.sh` classification agrees with the globs.
  - Evidence: protocol § 4 *The live gate* now defines a code PR by the same globs, gives the `implemented-by:`/`reviewed-by:` line format and the record comment forms, and notes that `task merge:check` also reads the ruleset and `class:flake` issues. These globs match `gateIsDoc` (`internal/review/gate.go:709-717`), which the script runs. § 2 step 5 now ends with the stop rule.

## 3. Shrink

- [x] 3.1 Shrink `AGENTS.md` L72, L82, L84, L89, L90 and L91 (D1), and the Close bullet's mechanics (D2), to one sentence each plus a pointer.
  - Acceptance: the row count is unchanged and `ateam doctor --check` is green. A trace table maps every removed clause to its retained AGENTS clause or its named section.
  - Evidence: 25 rows before and after; `ateam doctor --check` exits 0. Trace of the removed clauses:
    - L72: the four user-scope file names and `hook_sources` → `docs/adoption.md:38`. The project hook files and `ATEAM_HOOK_SCOPE` stay.
    - L82: the `ateam.json` key list → `docs/adoption.md` § Team settings. The gate-weakening, `ATEAM_*` and `.ateam/` rules stay verbatim. Adoption does not state the env-var rule, so it was kept.
    - L84: "≤ 60 lines: why, what changes…, `spec:check` warns" and the `<name> (<YYYY-MM-DD>)` form → protocol § 2 step 1 (`protocol.md:30`). The one-page rule, owner-only approval, the hold and the stop-and-hand-off rule stay.
    - L89: the code-PR globs, the agent list, the effort-parentheses form, the record comment forms, the ruleset and the flake check → protocol § 4 (added in 2.1). The two lines, reviewer strength, `cross_provider`, record freshness and `task merge:check` stay.
    - L90: running on `pull_request` from the PR's own tree, judged by its own copy, the fail-closed property, and "a later push fails it again" → protocol § 4 *Gate files* (`protocol.md:93`). The gate-file list, the label and comment, owner-only application and hand-off stay.
    - L91: "required check green on an up-to-date branch", "GitHub or Gitea", "the `main` run is informational" and review-checkout removal → protocol § 2 step 5. The worktree stop rule was added here.
    - Close (L101): the `spec:queue` rationale, the `gh` flags, the Gitea payload, the prune commands and review-checkout removal → protocol § 2 step 5. The stop rule and the red-job waiver stay.
- [x] 3.2 Record `wc -c AGENTS.md` before (`39917f4`) and after, overall and per row.
  - Evidence: 15,027 → 13,496 bytes (−1,531, −10%). Per row, before → after: L72 446 → 321, L82 495 → 415, L84 519 → 404, L89 958 → 557, L90 798 → 549, L91 421 → 339, Close 1,160 → 681. The design estimated about 2.4 KB; the actual figure is lower because the gate-file list, the hook file names and the env-var rule stay verbatim. `.agents/protocol.md` grows 24,627 → 25,329 (+702), read only by sessions that open § 2 or § 4. Neither figure is a measured token or productivity gain.

## 4. Verify

- [x] 4.1 `task verify`, `ateam bench --selftest`; archive before ready.
