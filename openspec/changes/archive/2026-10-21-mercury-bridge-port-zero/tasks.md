# Mercury port-zero fix — #104

## Delivery
- [x] Inspect the reported failure and pinned native CLI source; independently review the bounded port-zero plan.
- [x] Prove the startup regression before fixing it (nonzero reserved port; foreign listener received a readiness request and was accepted); update bridge startup and the affected fixture.
- [x] Prove the fix with three detected overlay mutations, 76 streaming split histories, focused race tests, and pinned LiteLLM offline startup (2.55s, zero provider requests). Spec attribution and lint pass.
- [x] Complete final-head full verification and doctor, obtain independent implementation review and the required cross-agent record, and archive this change for the authorized merge with Closes #104.

## Delivery evidence

The implementation at `18ea66580464a075068db3e46949e8c923cdc565` passed full local `task verify`, doctor (4.195ms, 183 schema tokens), [CI](https://github.com/C360Studio/ateam/actions/runs/39656904110), independent Codex review, and [commit-bound Gemini approval](https://github.com/C360Studio/ateam/pull/106#issuecomment-6324289110) with no findings. The reviewer also detected removal of the one-shot announcement guard.

The archival commit changes only Markdown. Test references already point to the permanent `openspec/specs/benchmark-experiments/spec.md` requirement, so no code or citation changes are needed. PR #106 closes #104 on merge; then the approved report closeout PR #103 can resume. Final-head checks and merge state are recorded on those PRs.
