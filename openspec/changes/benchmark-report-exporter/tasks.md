# Report exporter claim — #89

## Claim and design
- [x] Save the Mercury worked example through PR #85 before starting this claim; keep raw evidence private.
- [x] File focused implementation issue #89 under existing roadmap #16 and create a dedicated latest-main Codex worktree.
- [x] Independently review the exporter CLI/schema/trust boundaries; bind final-head documentation verification and review evidence to the draft-PR delivery.

## Implementation
- [ ] Add failing behavior tests and spec attribution for projection, partial/unknown accounting, replay identity and filesystem boundaries.
- [ ] Implement shared validated snapshot/projection and deterministic JSON/Markdown/manifest rendering without new statistics or model calls.
- [ ] Add safe export creation, optional replay checks and no-write drift-check mode through the existing CLI.
- [ ] Add public synthetic success/failure worked fixtures, repository check entrypoint and usage documentation; validate against retained Mercury/Gemini examples locally where available.
- [ ] Prove relevant tests with mutations/generated histories, run full verification and obtain independent final implementation review before delivery.

## Boundaries

This documentation claim does not implement the command. No live runs, UI/cloud storage or automatic publishing are part of #89. Existing model-run permissions are consumed; this offline implementation requires none. Merge remains subject to normal owner/cross-agent policy.
