# Verification evidence

Base: `10314ea0f4885343c2d0829b48be6f77fd016da4`; implementation snapshot is the commit containing this file. Local logs are in `.enjoy-logs/`. No project model-review policy is configured, so independent code review is not a policy-bound approval or proof of model superiority.

## Focused checks

- Before implementation, `go test ./internal/web -run TestModelRoutingStates -count=1` failed because the endpoint returned HTML instead of routing JSON and lacked no-store. Initial sandbox attempt could not access Go cache; the authorized execution produced the meaningful failure.
- `go test ./internal/web -run TestModelRouting -count=1`: passed, including state, authorization, confined-file and refresh-history coverage.
- `npm --prefix ui run test:unit -- src/lib/ModelRoutingPanel.test.ts`: 7 passed. `task ui:check`: zero errors/warnings. `task ui:build`: passed.
- `bin/ateam doctor --check`: passed (177 schema tokens, 4,181us observed hook latency).
- Independent reviewer `review_routing_ui` reviewed the complete change without editing it. Found a FIFO replacement blocking edge; owner reproduced the failure, fixed it with nonblocking confined open and descriptor validation, and added a deterministic regression. Reviewer re-reviewed and reported no remaining findings, independently rerunning the focused Go routing tests and diff check. Reviewer also ran the seven panel tests before the margin-only styling adjustment.
- Browser preview on localhost:8421 rendered the real unconfigured state, setup guidance and honest-intent note. Rendered component tests cover the other four states and escaped metadata. No project policy was installed for preview purposes.

## Testing decision

Decision: deterministic fixtures and an explicit history; no new fuzz generator.
Rule: only valid and eligible policy may be labeled eligible; higher reasoning effort cannot waive superiority; absent/unreadable/invalid policy must not imply eligibility; file replacement cannot block on a FIFO.
Inputs: finite status fixtures, same/different provider constraints, arbitrary/inherited effort, exactly 1 MiB and one byte over, directory/symlink/escaping path, auth/method matrix, and eligible -> blocked -> invalid -> deleted -> eligible history checked against literal expected states. FIFO replacement is synchronized between the path check and descriptor reader and uses bounded cleanup.
Run: focused commands above; no fuzz exploration claimed. The existing review parser retains its own tests; this change delegates grammar and ranking to it.
Mutation: `task mutate:check -- -pkg ./internal/web -test TestModelRoutingStates -file internal/web/routing.go -mutant /tmp/ateam-routing-mutant.go -expect-text 'routing status = eligible, want blocked'` detected incorrectly labeling blocked routes eligible. An initial anchored test-name invocation was inconclusive because the mutation tool expects a declaration name; the corrected exact-name run detected the mutation.
Mutation: `task mutate:check -- -pkg ./internal/web -test TestModelRoutingReplacedFIFO -file internal/web/routing.go -mutant /tmp/ateam-routing-fifo-mutant.go -expect-text 'policy replacement blocked on a FIFO'` detected removing nonblocking open. Both Go mutation checks passed baseline and after-run without changing repository sources.
Mutation: a temporary component mutation changed the blocked label to eligible; two UI assertions failed. Restored the byte-identical source from a copy (verified SHA-256), then all seven tests passed.
Not covered: native runtime model identity, policy editing, generated route-card drift, PR evidence/exception validation, mobile browser interaction, and FIFO behavior on non-Unix targets.

## Full preflight

The first `task verify` attempt passed formatting, spec shape/attribution, tidy, vet and lint. All Go packages except `cmd/ateam` passed with race detection. `TestCLISubcommands` hit the existing doctor timing gate at 52,809us against its 50,000us threshold during concurrent package verification. This failed attempt is preserved in `.enjoy-logs/routing-preflight-dirty.log`; it is not a passing verification result. The isolated CLI replay passed, and the remaining benchmark/UI checks exited 0 in `.enjoy-logs/routing-remaining-checks.log`: selftest passed with six evaluated checks and one explicitly unsupported task, UI checks reported zero errors/warnings, all 15 Vitest tests passed, and UI build passed. A clean-commit `task verify` is required before final delivery; final exact-SHA results are reported in the conversation and `.enjoy-logs/routing-preflight-final.log`.

## Publication boundary

Automatic approval review rejected creating the GitHub tracking issue because external publication was not explicitly authorized. No issue, push or PR was attempted afterward. The remaining publication task records this hold; local implementation and verification continued.
