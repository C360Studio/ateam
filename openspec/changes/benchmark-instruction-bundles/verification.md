# Implementation verification

implemented-by: codex (bundle resolver, retained evidence, shared transport integration, offline tests and examples)

Addresses #26 under roadmap #16. Base: 76e6f63d8deae62c075923f91fb703afe2d6c5a5. Branch: codex/benchmark-treatments. The initial claim-only checkpoint was 8e86f466c80b2c4b07bd62ab8d47cb5fdcb27399; the owner subsequently authorized implementation and delivery.

No `.agents/review-policy.json` is configured at this base or in this change. Independent plan and substantive change review use a separate read-only agent run that authored none of the deliverable. Runtime model identity/superiority is not independently attested; no configured-policy approval is claimed. The reviewer approved the schema and bounds and found a relative-config-directory issue during code review. A failing changed-cwd regression reproduced it before the correction. A second failing regression reproduced rejection of semantically identical native settings after JSON indentation; the corrected guard retains full provenance comparison and compares native settings semantically. Tests also reject actual native-settings changes. Final committed-head review and full clean-tree verification are delivery gates, recorded against their exact SHAs in PR #27. The first full gate at `26b631f08d584c32df3cddb00b9cca8aceb517bc` stopped at strict OpenSpec validation: pinned local 1.14.3 rejects requirement descriptions over 500 characters, including three pre-existing dashboard/routing descriptions. Exact normative details were moved into scenarios without changing requirement anchors or behavior. The previously passing direct check used global OpenSpec 1.7.0; it was not equivalent to the pinned full gate.

## Focused offline evidence

- The two tracked bundle examples resolved offline and each completed six reference-adapter attempts. Comparison and rescore outputs matched; original artifact bytes stayed unchanged. The reference adapter ignores instructions, so these are pipeline checks, not performance evidence.
- Historical real Haiku and Luna pilot artifacts passed read-only load/compare using the extended implementation. SHA-256 values stayed `9c68c17cfcff2c1673266f0d7546b26bdf37cc68f4e22d593eabaf549803f0c8` and `ce5fcbe681d2279fd66c999ea38e0e8a64f7759e1454976f72dc00626c428873`. Neither raw artifact was copied into this worktree or published.
- `TestBundleDeliveryToGenericTransport` and `TestBundleDeliveryToNativeTransports` passed all eight arm/transport cases, with literal Unicode/CRLF/quotes/backslash/whitespace oracles. Existing three native isolation regressions passed. Native tests use fake CLI processes, not provider calls.
- `task test:spec` passed after adding directly attached citations. Strict OpenSpec validation and `git diff --check` passed.

## Testing decision and mutation evidence

Decision: Generated composition checks plus deterministic parser, boundary and state-history examples. No generated filesystem-race claim.

Rule: Preserve ordered component bytes and two-newline separators; use effective bytes for AA/AB; confine and bound explicit inputs; validate retained evidence without source access; refuse changed inputs/controls before execution.

Inputs: Three fuzz seeds include empty content, Unicode/BOM/CRLF and 32 components. Examples cover metadata-only equivalence, alternate component splits, reversed order, manifests at 64 KiB and one byte over, 32/33 file paths, effective content at 1 MiB and one byte over, 600-attempt JSON-escaped artifact expansion, missing files, directories/symlinks/traversal, changed content/metadata/argv/provenance, corruption under a recomputed outer digest, and resolve/change-cwd/run/delete-sources/compare/rescore history. Literal source/hash expectations and the test-owned byte-append model supply independent oracles.

Run: `go test ./internal/bench -run '^$' -fuzz '^FuzzBundleComposition$' -fuzztime 10s -parallel 2 -v` passed 543,747 generated executions after replaying all three seeds; this is exploration, separately from ordinary seed replay. Logs and generated cache are local and ignored. These focused checks measured the implementation snapshot before final commit; final repository checks are bound to committed heads in the PR.

Mutation: Five non-destructive `task mutate:check` overlays reported `detection`, each with passing baseline and after-run: select baseline bytes for the candidate (all three native transports); replace the two-newline separator with one; omit component hash verification; omit known artifact-size refusal; and omit the whole resolved-bundle stale check (content, metadata, argv, policy, runner and runtime mutations were then accepted). Tests were `TestBundleDeliveryToNativeTransports`, `TestBundleImmutableResolution`, `TestBundleTamperedSnapshotRefusal`, `TestBundlePredictableArtifactLimit` and `TestBundleStaleResolutionNeverLaunches`. Mutants were temporary copies outside the repository; original source files were never swapped or restored. There are no known survivors.

Not covered: Paid model calls, Gemini capacity retry, native skill/hook execution, provider billing reconciliation, adversarial concurrent edits to already-open file contents, and exhaustive platform-specific filesystem behavior. Known request sizing cannot predict response size. Retained hashes establish consistency, not authenticated authorship. No harness-performance improvement is claimed.

## Completed delivery gates

At `032b84e58c2955129b42bcfd35831464dcb73511`, `GOFLAGS=-p=1 task verify && task doctor` exited 0 on a clean tree. This ran formatting, pinned OpenSpec validation, test attribution, tidy, vet, lint, every Go package under race, offline benchmark self-tests, Svelte checks, all 23 UI tests and the UI build. Doctor reported parity passed and 177 MCP schema tokens. An independent read-only review approved the complete base-to-head code/spec/example diff with no remaining blockers; both reported correctness findings were fixed and tested. The final documentation checkpoint and its exact verification/review binding are recorded in PR #27.

The earlier default-parallel full run at the same code revision failed the existing `TestConfiguredGitHubTarget`: the fake CLI log contained fallback and PR calls but lacked the initial comments call. Its first invocation has a 1.5-second deadline; the observation is consistent with process-start contention, not proof of the cause. The unchanged test then passed 5/5 isolated race repetitions, and the entire package-serialized race suite passed. No state implementation or timeouts were altered. This verifies serialized coverage, not stability of the default parallel run; that timing sensitivity remains a disclosed limitation.

Skipped: paid calls, Gemini retries and performance claims. Residual risk: the existing parallel forge-test timing sensitivity and the explicit filesystem/artifact limits above.

## Main integration

Main advanced to `5c005f4bd859f5532508067d9aa0ecdc30cefbfb` (MCP registration and client integration) during delivery. Its four dashboard/routing spec formatting corrections overlapped our equivalent strict-validator fixes. The merge keeps main's exact versions of those four files and all sister-agent MCP changes; benchmark implementation is unchanged. Final post-merge verification and independent review are bound to the integration head in PR #27.
