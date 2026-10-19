# Benchmark verification evidence

## Scope and reproducibility

Implementation is confined to the benchmark package/CLI, example adapter and guide, local OpenSpec tooling, and OpenSpec artifacts. Main README and UI sources are untouched. The harness spec was reformatted without changing behavior because OpenSpec 1.14.3 strict validation rejects its existing oversized webhook requirement.

OpenSpec is pinned to 1.14.3 in the root npm lockfile. Installed global 1.7.0 remains unchanged. Use `npm ci` for local tooling and `npm ci --prefix ui` for the existing UI lockfile before full verification.

The initial sandboxed `task verify` failed on two distinct causes: a test helper used a 1-second deadline while the race runtime waits at exit, and an existing HTTP integration test could not bind a loopback port. The adapter test now uses 5 seconds for ordinary completion/output-limit cases and retains 1 second for the actual deadline case; its focused race regression passes. The full rerun was approved with loopback access. This is not a skipped integration test.

## Behavioral checks

The offline example exercises 3 tasks × 2 instruction arms × 2 repetitions. All 12 reference attempts pass, usage remains unknown, and offline rescore succeeds. This validates plumbing only: the adapter returns fixed reference sources and performs no model calls.

Tests cover strict config and complete-history validation, counterbalanced schedule, script provenance, isolated HOME/environment, missing/invalid usage, malformed/oversized responses, deadlines, Unix process-group cancellation, candidate-versus-tool failure classification, non-overwriting saves/rescore, final-grade cancellation, and descriptive paired comparison.

The independent plan review requested explicit task contracts, script provenance, and grader isolation. The independent change review found four concrete issues (final-grade cancellation, package mismatch, infrastructure-versus-quality wording, actual grader identity); regression checks were added and all findings corrected. The reviewer approved the scoped offline pilot without editing implementation files.

## Six-line testing record

Decision: generated JSON decoding plus deterministic tables/history tests; no probabilistic performance claim.
Rule: unknown fields/invalid limits/duplicate tasks are refused; every configured pair occurs exactly once; incomplete or contradictory evidence is refused; failed candidates cannot improve successful-pair time; cancellation cannot create a successful completed rescore.
Inputs: 3 fuzz seeds, repetition boundaries 0/1/100/101, unsupported and duplicate tasks, unknown/trailing JSON, token boundaries, malformed/oversized output, wrong packages, missing test execution, invalid cache/tooling, matched/mismatched artifacts, and explicit child readiness signals.
Run: deterministic seed replay is included in unit tests; separate exploration `GOCACHE=/private/tmp/ateam-bench-audit-cache go test ./internal/bench -run '^$' -fuzz '^FuzzExperimentDecode$' -fuzztime=3s -parallel=1` passed 94,785 executions (3 initial seeds, 55 new interesting inputs) before final metadata-only refinements. Focused race tests and final full verification cover the retained seeds on the final implementation.
Mutation: six intentional wrong behaviors produced detection using non-destructive Go overlays; details below. No survivor, invalid, or inconclusive result is claimed as detection.
Not covered: live providers/native adapters, paid usage reconciliation, Windows descendant cleanup, adversarial OS containment, representative repository tasks, statistical significance/equivalence, timing side channels, or end-to-end agent skill/tool adoption.

## Mutation evidence

All commands use `GOCACHE=/private/tmp/ateam-bench-audit-cache task mutate:check -- -pkg ./internal/bench ... -runs 1`; baseline/expected-mutant-failure/after-run protocol passed.

| Wrong behavior | Test | Target SHA-256 at mutation check | Verdict |
| --- | --- | --- | --- |
| Continuity invents bad/mutant success | `TestContinuityDoesNotClaimMutationDetection` | `70505a146c47769d9ec02a55380b6684e98b6ac057702fd8769c11984a4c59d2` | detection |
| Candidate passes without an executed trusted test | `TestCandidateRequiresExecutedTrustedTest` | `70505a146c47769d9ec02a55380b6684e98b6ac057702fd8769c11984a4c59d2` | detection |
| Wrong-package candidate treated as infrastructure | `TestCandidateEvaluationOutcomes/wrong-package` | `6da4a5a43d0a7c559c37e53b76760f76867f32d3ee5ce782a66abda13aa65b59` | detection |
| Duplicate tasks accepted | `TestExperimentValidationAndSchedule` | `8b6283b5cdfb0658e461bffcd925d30445accbbc52fd1fe79601a034d633f97b` | detection |
| Failed candidate included in successful-pair timing | `TestComparisonOracleAndRefusals` | `8645800bee8698887cd117ba9bc3669706a76e913d01d80be10381a0dc329740` | detection |
| Cancellation in last grade returns success | `TestRescoreCancellationDuringFinalGrade` | `34dc0c9656db99d22d63bd2ebde966b9455881080f9980631e8f932ca97b430f` | detection |

The first two checks preceded the wrong-package classification fix; the first three preceded the final report-only “estimated tokens” wording correction. These are explicitly recorded source snapshots, not claims that every mutation was rerun after unrelated changes. All regression tests remain in the full race suite.

Parent mutation logs are in ignored `.ateam/mutation-{config,comparison,cancel}.log`. Temporary mutant copies are `/private/tmp/ateam-bench-mutant-{config,compare,cancel}.go`, `/private/tmp/ateam-bench-{continuity,grader,package}-mutant.go`. Recreate copies with those single intentional wrong behaviors to reproduce after temporary-file cleanup.

## Limitations

Trusted local execution is not a security sandbox. Non-Unix cleanup only terminates the direct child; usage is adapter-reported; three small public tasks support a pilot, not general agent-performance claims. The existing UI dependency installation reported 6 advisories (3 low, 1 moderate, 2 critical); that pre-existing lockfile was not upgraded in this benchmark scope.

## Full-gate result

`GOCACHE=/private/tmp/ateam-bench-audit-cache task verify` passed with exit 0 on clean commit `0d5da3174fb829739e199a1f90c5e6ad173324d0`, with approval for existing loopback HTTP integration tests. This included formatting, local OpenSpec validation, tidy, vet, revive, all Go tests with race detection, offline self-tests, Svelte checking/unit tests/build, and the final clean-tree check. Log: `.enjoy-logs/benchmark-verify-2.log` (ignored local evidence). `go run ./cmd/ateam doctor` also passed.

Subsequent archive/evidence changes are documentation only. Final clean-HEAD verification is recorded in the draft PR after the archive commit rather than rewriting this evidence into a different source snapshot.
