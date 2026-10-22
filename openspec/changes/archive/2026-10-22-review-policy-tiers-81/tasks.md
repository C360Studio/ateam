## 1. Implementation

- [x] 1.1 Parse and validate `model_tiers` and `effort_bands`; make dormant-path fields optional when no models are declared (#81).
- [x] 1.2 Fail closed in `review route` / `review check`; no route cards from setup or doctor; dashboard `unconfigured` for a tiers-only policy (#81).
- [x] 1.3 Commit `.agents/review-policy.json`, add the spec requirement and tests, and record mutation detection for the duplicate-tier check, the provider check, the setup, dashboard and route guards (#81).
- [x] 1.4 From the Codex review of `b44186d`: omit empty optional fields so a tiers-only policy round-trips (seeded in `FuzzPolicyJSON`), and validate supplied effort bands even without tiers (#81).
