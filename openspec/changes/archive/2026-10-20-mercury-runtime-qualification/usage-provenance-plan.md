# Offline provider-usage provenance slice

Addresses #58 and #16; follows merged PR #64 (`03121ae8066ec58a2ecc5a95c38c4447389eb0a3`). This new claim is `codex/mercury-usage-provenance`, based on that main commit. The previous worktree is clean and its PR merged; no other open PR claims Mercury. PR #29 still owns native skills. The owner requested implementation through review readiness, not merge or paid calls.

## Scope and decision

Deliver a pinned-source audit of supported LiteLLM usage controls, and the smallest offline evidence contract that can retain provider usage before normalization. Prefer supported behavior over a bridge patch. If the audited controls cannot establish provenance, use the existing retained synthetic provider responses to demonstrate a separate, opt-in, stdlib-only evidence reader. Do not add a network proxy, fork LiteLLM, replace the native runtime or wire a production adapter in this slice.

Callers and files in scope: this plan, `tasks.md`, a usage-provenance report, new offline reader/tests, existing `offline/observations.json` as read-only replay input, and `offline/check_sentinel.py` for the approved PR #64 cwd-precondition nit. Existing `reproduce.py`, stack/lock, native bench packages/configs, historical artifacts and frozen Gemini evidence stay unchanged. Inspect installed pinned package source as data; no new package installation or imported dependency code is needed.

## Acceptance and verification

1. Audit `stream_options.include_usage`, always-include usage, raw-response/callback paths and fallback counting at the pinned 1.99.0 source. State inspected candidates and limits, without claiming absence of every possible solution.
2. Specify provider-wire presence independently from translated estimates, transport completion and primary/detail completeness. Missing values remain unknown; explicit zero remains zero; raw cached/reasoning subsets are retained without double counting. Refuse malformed, inconsistent or ambiguous evidence.
3. Demonstrate the contract on PR #64's supplied/missing-usage observations and deterministic synthetic cases, including incomplete streams, partial fields, zero, duplicates/conflicting identities, failed/retried requests and cancellation. Every exchange remains separately accounted for. The prototype cannot authenticate a future capture source or prove no unobserved request occurred; production capture/correlation stays a gate.
4. Use independent expectations, generated bounded combinations/history permutations and non-destructive copied-source mutations. Prove the cwd nit with a failing out-of-home regression before adding its guard. Run explicit Python checks (ordinary Go/Vitest CI does not execute them), strict OpenSpec validation, clean-tree `task verify`, and doctor.
5. Obtain independent read-only plan and exact-head technical review. No configured stronger-model policy exists; disclose unavailable reviewer identity and leave owner-arranged cross-service approval for merge.

## Stop boundary and risks

Stop at a review-ready PR containing the audited, tested evidence contract or a concrete blocker. A passing offline reader is not runtime qualification. Real capture correlation, all requests/retries, tool permissions, cancellation, errors and live provider semantics still require native-runtime evidence. No price, cost or performance claim; no paid call, production adapter, new runtime or automatic continuation into calibration. Keep #58 open and its qualification/spending holds active.
