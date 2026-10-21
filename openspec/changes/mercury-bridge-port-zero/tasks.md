# Mercury port-zero fix — #104

## Delivery
- [x] Inspect the reported failure and pinned native CLI source; independently review the bounded port-zero plan.
- [x] Prove the startup regression before fixing it (nonzero reserved port; foreign listener received a readiness request and was accepted); update bridge startup and the affected fixture.
- [x] Prove the fix with three detected overlay mutations, 76 streaming split histories, focused race tests, and pinned LiteLLM offline startup (2.55s, zero provider requests). Spec attribution and lint pass.
- [ ] Complete final-head full verification and doctor, obtain independent implementation review and the required cross-agent record, archive this change and merge with Closes #104; resume PR #103 afterward.
