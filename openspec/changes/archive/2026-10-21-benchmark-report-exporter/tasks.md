# Report exporter claim — #89

## Claim and design
- [x] Save the Mercury worked example through PR #85 before starting this claim; keep raw evidence private.
- [x] File focused implementation issue #89 under existing roadmap #16 and create a dedicated latest-main Codex worktree.
- [x] Independently review the exporter CLI/schema/trust boundaries; bind final-head documentation verification and review evidence to the draft-PR delivery.

## Implementation
- [x] Add failing behavior tests and spec attribution for projection, partial/unknown accounting, replay identity and filesystem boundaries.
- [x] Implement shared validated snapshot/projection and deterministic JSON/Markdown/manifest rendering without new statistics or model calls.
- [x] Add safe export creation, optional replay checks and no-write drift-check mode through the existing CLI.
- [x] Add public synthetic success/failure worked fixtures, repository check entrypoint and usage documentation; validate against retained Mercury A/B plus replay locally (12/12, 30,947 tokens, matched replay, unchanged originals); Gemini native partial-usage regression uses synthetic evidence because no retained Gemini run was found in available worktrees.
- [x] Prove relevant tests with mutations/generated histories: unknown-to-zero, private projection, replay association, escaping, exclusive file creation and CLI destination guards are detected; 81 filesystem histories and 21,717 final rendering-fuzz executions pass (60 cached seeds replayed separately; earlier exploration completed 32,342 executions).
- [x] Record full final-head verification and independent implementation review in PR #90; satisfy the separate Claude/Gemini commit-bound review gate before changing this code PR out of draft.

## Boundaries

No live runs, UI/cloud storage or automatic publishing are part of #89. Existing model-run permissions are consumed; this offline implementation requires none. Merge remains subject to normal owner/cross-agent policy.

The output parent must be trusted/stable through initial creation/open. Post-anchor replacement is detected; portable Go cannot prove ownership across an adversarial mkdir-to-first-identity gap. This explicit limitation replaces the initial broader race-safety wording.

## Completion evidence

Implementation merged through [PR #90](https://github.com/C360Studio/ateam/pull/90) at `6ef534c8971441d1941454a6440ab21fa68cf499` on 2026-10-21. The reviewed implementation head was `eade7bebe5a9eebacd021b6f8d0e868a2e30aa2b`.

- Full local `task verify` and doctor passed at the reviewed head; the final readiness [CI run](https://github.com/C360Studio/ateam/actions/runs/39642849424) passed Verify, Merge check, and Required before merge.
- [Independent Gemini approval](https://github.com/C360Studio/ateam/pull/90#issuecomment-6321455129) covered that exact head. The non-draft commit-bound review gate passed.
- The three low, non-blocking review observations were accepted for this delivery, not silently fixed: an optional direct unknown-row assertion (covered by sibling tests), explicit empty replay treated as omitted, and preserved legacy Gemini comparison accounting alongside report accounting.
- Current requirements are promoted to `openspec/specs/benchmark-reports/spec.md`; this directory retains the historical proposal, design, delta and completed checklist.
