# Implementation verification

Scope: issue #14 / PR #15, model routing and independent review policy. Current main was merged before implementation. Final clean-commit full verification and independent review results are recorded in the PR; this file records reproducible implementation-specific evidence.

## Behavior and independent review fixes

The gate requires explicitly stronger models for all implementers, separate independent runs, authorized record authors, current head/base/policy binding and reviewer-executed configured checks. Scoped owner exceptions waive only superiority. The independent Astra review found three defects before approval: inaccurate Gitea PR-comment fixtures, arbitrary target-branch policy authority, and editable-comment provenance. Failing regressions were demonstrated before fixing provider-specific parent URLs, verified default-branch metadata/SHA and known-edit refusal plus exact evidence rereads. Equal-timestamp edits remain an explicit attestation limitation.

## Generated checks

- Decision: generated checks used for ordering, combinations and histories; deterministic HTTP fixtures cover each forge boundary and cancellation with explicit synchronization.
- Rule: approval requires current trusted scope, independent eligible models, authorized evidence, required passing checks and valid scoped exceptions.
- Inputs: 649 generated integer-rank comparisons (equal, adjacent, transitive, ceiling); 256 four-step approval histories (1,024 assertions) against a test-owned freshness model; all nine native-provider route pairs; exact 256-model and 1 MiB JSON boundaries.
- Run: `go test -v ./internal/review -run 'TestGeneratedRankComparisons|TestReviewHistory|TestPolicyModelLimit|FuzzPolicyJSON'` replays deterministic checks and fuzz seeds. Separate exploration: `go test ./internal/review -run '^$' -fuzz '^FuzzPolicyJSON$' -fuzztime=3s -parallel=1` completed 1,999 executions in the initial run; generated assertions log completed counts rather than skipping empty ranges.
- Mutation: ten non-destructive Go-overlay detections listed below; baseline and after-run passed for each. No production file was mutated in place.
- Not covered: cryptographic model/runtime proof, complete attribution of undisclosed implementers, same-timestamp comment edits, and live Gitea deployment testing. No paid model runs were used for benchmark validation.

## Mutation reproduction

For each row, copy the source outside the repository, apply only the indicated wrong change, then run:

```sh
task mutate:check -- -pkg PACKAGE -test TEST -file SOURCE -mutant /tmp/mutant.go -expect-text 'ASSERTION'
```

| Package / source | Test | Wrong change | Assertion | Outcome |
| --- | --- | --- | --- | --- |
| `./internal/review`, `policy.go` | `TestEligible` | Accept equal reviewer/developer IDs as stronger | `eligibility got <nil>, want allowed=false` | detection |
| `./internal/review`, `record.go` | `TestReviewHistory` | Ignore expected head mismatch | `fresh=false got <nil>` | detection |
| `./internal/review`, `record.go` | `TestOwnerException` | Ignore exception head binding | `unscoped exception approved` | detection |
| `./internal/review`, `record.go` | `TestRecordAuthor` | Bypass review-author allowlist | `unauthorized record author` | detection |
| `./internal/bootstrap`, `review.go` | `TestConfigureReviewConflictNoWrites` | Bypass existing-file conflict refusal | `conflict must fail before writes` | detection |
| `./internal/bootstrap`, `bootstrap.go` | `TestReviewSetupDrift` | Remove review setup from parity checks | `expected violation containing` | detection |
| `./internal/review`, `forge.go` | `TestForgeRejectsUnboundComments` | Disable `parent != expectedParent` | `unbound comment accepted` | detection |
| `./internal/review`, `forge.go` | `TestForgeTargetAuthority` | Disable `ref != repo.DefaultBranch` | `target authority: success=true` | detection |
| `./internal/review`, `forge.go` | `TestForgeRejectEditedComments` | Disable `!created.Equal(updated)` | `edited or unprovable comment accepted` | detection |
| `./internal/review`, `repository.go` | `TestBaselinePolicyRejectsCandidateAuthority` | Load policy from head instead of base | `candidate controlled governing policy` | detection |

Source paths in this table are relative to their package directory. Mutations measured these production source bytes (documentation-only changes do not alter this evidence):

| Source | SHA-256 |
| --- | --- |
| `internal/review/policy.go` | `f35171d027b06215103fe48415ef96a74cdeee567c827a6f3d398226ff91ee11` |
| `internal/review/record.go` | `9b08a6fc2907b81815b4d0b3218c4e3b59b1fa27e42fec0c6bf6a12ab3faf0f0` |
| `internal/review/forge.go` | `d20639615080225bc56bc8e020ae24a05a9ee1f6a876bc9354e24bc786ca3cc5` |
| `internal/review/repository.go` | `ae943f26372f00d3b3b2432c03ff9a713dc879aa6eeb02753b14cfe178b3fd50` |
| `internal/bootstrap/review.go` | `9c533baae262d5de6dbddb4ee33cead5e2b3307427361cf0bfabc6cf2af9874a` |
| `internal/bootstrap/bootstrap.go` | `22febf1abe1064f981cc1db0b41c2d3d5d6c22ef9e3b499de326c26508036062` |

## Validation limits and resolved environment issues

The original planning claim lacked UI dependencies; lockfile-pinned root and UI dependencies are now installed with install scripts disabled. Initial sandboxed Go cache/loopback restrictions were resolved through provider-approved test execution, not bypasses. One concurrent test run saw old transport fixtures while the authority regression fix was being written; it was a failed run, followed by passing focused regressions. Final full gates must use the frozen commit.

Installation reported six existing UI dependency audit advisories (three low, one moderate, two critical); no dependency versions were changed by this feature. Live forge/model availability is not certified by local fixtures. This change does not install branch protection or merge automatically.

## Frozen implementation result

At `8775c08eedbdcbbf2aca56519abbbca4ba0968b1`, `task verify` exited 0 on a clean tree, including fmt/spec/tidy/vet/lint, all Go race/unit tests, benchmark self-tests, UI type checks/tests/build and the final clean-tree check. `task doctor` exited 0 with no violations and 177 MCP schema tokens; its one hook sample was 5,038 microseconds, so the passing doctor result is not a separate proof of the advertised 5ms threshold. Generated checks at this commit logged 649 comparisons and 1,024 history assertions, plus all four fuzz seed replays.

Changes through `e3c8c4eb525f93c5cb58744d96f8b086332ba30a` merge main's `.env` ignore and reconcile/archive these documentation artifacts; runtime sources were unchanged at that checkpoint. The final exact-commit independent reviewer run and verdict belong to PR #15. The reviewer is the requested Astra run and wrote none of the deliverable; model superiority against every contributor was not independently authenticated. This repository has no installed governing model policy, so the new strict routing gate itself is not claimed as satisfied by this feature's review. Project model comparisons and approved posting accounts must be configured before enabling that gate.

## Requested review follow-up

[The independent Gemini review](https://github.com/C360Studio/ateam/pull/15#issuecomment-6271003100) identified inactive provider pointers, missing no-policy contract checks, and token normalization/fallback gaps. Each was reproduced by failing tests before implementation. Setup now refuses inactive reserved paths before writing, doctor checks canonical contracts even without policy and reports orphan cards/pointers, and CLI credentials trim whitespace and fall back to `ATEAM_GITEA_TOKEN`. A separate failing compatibility regression prevented rejecting an unrelated linked provider directory without a reserved role pointer.

`go test -race ./internal/bootstrap ./cmd/ateam -run 'TestConfigureReview|TestReview' -count=1` passed. Generated checks exercise all nine provider pairs and explicitly assert 24 stale-pointer/reconciliation histories, using expected reserved paths independent of production enumeration. Examples cover each contract deletion/drift, orphan/dangling pointer and token precedence boundary; arbitrary token fuzzing adds no useful oracle beyond these normalization cases.

Four additional non-destructive overlay mutations each produced `detection`, with passing baseline and after-run checks. Reproduce using the command above and these source-relative changes:

| Package / source | Test | Wrong change | Assertion | Outcome |
| --- | --- | --- | --- | --- |
| `./internal/bootstrap`, `review.go` | `TestConfigureReviewConflictNoWrites` | Disable the inactive-path conflict condition | `conflict must fail before writes` | detection |
| `./internal/bootstrap`, `review.go` | `TestReviewInactivePointers` | Initialize doctor violations without `unexpectedReviewFiles` | `expected violation containing` | detection |
| `./internal/bootstrap`, `review.go` | `TestReviewUnconfiguredArtifacts` | Return early when policy is nil | `expected violation containing` | detection |
| `./cmd/ateam`, `review.go` | `TestReviewTokenEnvironment` | Return an empty token instead of the Gitea fallback | `token normalization/fallback mismatch` | detection |

These follow-up mutations used `internal/bootstrap/review.go` SHA-256 `2c46aa59406e98583f6067a4ae8368a2beaf199249896eef94eac134180c202c` and `cmd/ateam/review.go` SHA-256 `805b5b56ba924cd408fb7e0c25a318451048af541c94bf053e165ee97d77835d`. Final clean-commit gates and refreshed independent approval are recorded on PR #15. Earlier evidence above remains attributed to its original source bytes.
