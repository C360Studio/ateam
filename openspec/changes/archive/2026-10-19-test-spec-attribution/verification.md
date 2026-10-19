# Verification: test-to-spec attribution

implemented-by: codex

Implementation checkpoint: `55fcda6e9f4c6763d11c502aa27e4d629aeb2081`, integrated with main `fc9adbf282bcd6c8cb76abe5b5b67e34147204eb`. The final completion commit changes task/evidence prose only; its full verification result is recorded on PR #19.

## Acceptance evidence

- `GOCACHE=/private/tmp/ateam-attribution-go-cache task verify` passed (exit 0) on the clean implementation checkpoint, with sandbox approval for existing loopback HTTP integration tests. It includes strict OpenSpec validation, test attribution and its Vitest fixtures, tidy, vet, revive, every Go package under the race detector, offline benchmark self-tests, Svelte checking, all 8 Vitest tests, static build, and clean-tree verification. Log: `.enjoy-logs/attribution-verify-1.log` (local ignored evidence).
- `go run ./cmd/ateam doctor` passed with no violations; the observed invocation reported 7,682 microseconds. Doctor currently enforces 50ms, so this is not evidence that every cold invocation meets the aspirational 5ms target. The existing hook latency regression passed in the full suite.
- The independent reviewer approved the complete diff against current main, including conflict resolutions, updated generated instructions, and model-review test annotations. The reviewer independently ran focused Go checker/CLI/wiring/bootstrap checks, the five scanner fixtures, TypeScript infrastructure regression, and whitespace checks. Mutation/fuzz results below were author-executed, not independently repeated.
- `.agents/review-policy.json` is absent, so the optional configured model-ranking gate is not active. Independent review was a separate read-only agent run, not author self-approval.

## Six-line testing record

Decision: Use a native Go fuzz target for exact requirement-title resolution, deterministic example fixtures for source AST/attachment/path boundaries, and a subprocess wiring regression for Taskfile fail propagation. No state-history model is required: this checker does not mutate repository state.

Rule: Every supported declaration needs an attached citation; every citation resolves exactly one allowed requirement; unrelated strings, suite comments, detached comments, unsafe paths, and parser errors must not permit false success.

Inputs: Explicit Go fixtures exercise missing/valid/malformed citations, comment gaps, fixture strings, TestMain/helpers, Fuzz, non-executable and whole-file executable examples, parse failures, missing/duplicate/fenced headings, traversal/absolute/arbitrary paths, symlink escape, ignored directories, build tags, testdata, deterministic order, and cancellation. Vitest fixtures exercise import aliases/namespaces/globals, each/condition/modifier chains, shadowing, unsupported aliases/imports/access, parser failures, and detached comment groups. The integration regression reaches the embedded scanner and proves missing TypeScript is an infrastructure failure. The wiring regression executes verify with a test-owned failing checker stub and proves later gates are not run.

Run: Focused regressions passed. `go test ./internal/testspec -run '^$' -fuzz FuzzRequirementResolution -fuzztime=3s -parallel=2` completed 12,879 executions with 5 initial seeds and 28 new interesting inputs; no failure. This was exploration, distinct from seed replay in `task test:unit`. Every accepted generated title runs both the exact-resolution assertion and a deliberately unresolved-title assertion; empty/non-UTF-8/multiline titles are filtered and covered by explicit refusal examples where applicable. Required boundaries are covered by deterministic cases rather than probability.

Mutation: `task mutate:check -- -pkg ./internal/testspec -test TestGoAttribution -file internal/testspec/check.go -mutant /private/tmp/ateam-attribution-missing.go -expect-text 'attribution verdict'` detected disabling the missing-citation branch (`if !found && len(declarations) < 0`). `TestReferenceRefusals` with `-expect-text 'reference refusal'` detected replacing `if count != 1` with `if count < 0`. Baseline and after-runs passed, source SHA-256 `772ab0f296f2a4bee1a221cfd8491f57a836d0c3515f3acbcb81b1d1e3b3e0f6` stayed unchanged. An initial unused-variable mutation did not compile and was recorded as invalid before using the compilable mutation. Disabling the scanner's comment-gap guard made exactly its gap fixture fail; backup/restore preserved SHA-256 `9eaccfb786e691c116f10af4095b031ffd83375fc4f6c748d9d52891ba0179bb`, then all five fixtures passed again. No unresolved survivors.

Not covered: Semantic proof that assertions establish the cited behavior, reverse spec-to-test completeness, arbitrary Vitest wrapper modules, runtime-expanded case counts, other frameworks, and unsupported filenames/languages. These are explicit adoption boundaries, not implied coverage. The scanner's own regression declarations are supported Vitest `.test.js` files and carry validated citations.
