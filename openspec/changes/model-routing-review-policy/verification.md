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
