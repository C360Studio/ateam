# Verification

implemented-by: codex, draft PR #13 (Addresses #10; user-authorized Claude/Haiku extension)

Independent plan and read-only change review approved the implementation with no blocking code findings. The reviewer required bare API-only auth, managed-policy handling, contradictory-terminal and identity refusal, and explicit missing-accounting semantics; these are implemented. Live policy preflight passed; the first telemetry pilot exposed a reply-format defect, described below.

- Decision: exhaustive generated counter presence/null combinations plus finite malformed-JSON, process and history scenarios. Claude-specific fuzz exploration is not included: the finite oracle covers each required counter, aggregate boundary and terminal/refusal branch; existing Gemini JSON fuzz seeds exercise the shared unique-key decoder. The finite models cover the current four mandatory token counters and deliberate stop cases.
- Rule: whole-call per-model counts include each cache category exactly once; top-level main-loop totals overlap and must not be summed; missing accounting stays unknown; only clean fixed-model terminal responses can be graded.
- Inputs: 32 generated counter cases, partial sibling models, exact 1e12 and aggregate limits, optional thinking, malformed/duplicate/trailing JSON, contradictory provider/model/terminal status, errors, missing auth, allowlisted environment, two fresh arms, output truncation, readiness-synchronized cancellation, and six reference histories (complete/token/unknown/model/error/reuse).
- Run: `GOCACHE="$PWD/.enjoy-logs/go-cache" go test ./internal/bench -run '^Test(Claude|Native|ExperimentMode)' -count=1` passed (19.071s). Claude examples resolve installed 2.1.292 and 30 attempts with no model calls. Existing Gemini capacity-failure artifact loads/compares under the new binary unchanged. `task fmt:check spec:check lint` passed. Doctor passed, 177 schema tokens and 4010 microseconds for its single hook observation.
- Mutation: detection for dropping cache creation from normalized input, removing --bare, and bypassing pre-grade model refusal; baseline/after passed and original hashes stayed unchanged. Direct Go overlays under ignored .enjoy-logs were used because the mutation command rejects in-worktree mutants while this session is constrained to this worktree. Logs: claude-events-mutation.log, claude-model-guard-mutation.log, claude-bare-final-mutation.log; execution mutation assertion was `execution policy arguments differ`.
- Not covered: successful live API run, authoritative billing, remote policy preflight, arbitrary skills/hooks, Windows registry inspection, or Codex Luna. CLI output is synthetic in tests, not a recorded live fixture. Native cost estimates are not hard spending caps. Existing repeated-session handling grades before detecting reuse and clears its grade timing; rejected attempts must not be used as successful execution timing.

Full clean-tree `GOCACHE="$PWD/.enjoy-logs/go-cache" task verify` passed with exit 0 at implementation commit `a1595818a69c1e18e647726f50fffe7a983b55fc`; log `.enjoy-logs/claude-verify.log`. This included full Go race tests, offline benchmark self-tests, format/tidy/vet/lint/OpenSpec and UI check/test/build. Final evidence-only checkpoint receives another clean-tree verify before push, with its SHA and outcome recorded in the PR description. This was the pre-live checkpoint; the credential is now present in the main ignored local environment file and is supplied to child processes only.


## Native structured-output correction

- Decision: finite contract cases cover exact property names/types, required native object, conflicting text, legacy replay and execution refusal; no additional fuzz exploration.
- Rule: new calls require native JSON schema and structured_output; old artifacts retain their text parser and exact arguments and cannot execute again. StructuredOutput may finish with tool_use; that stop reason remains invalid in legacy mode.
- Inputs: valid native object beside fenced/conflicting text, missing/null/extra/wrong-case fields, incomplete terminal reasons, legacy argument/provenance preservation and the actual failed live artifact.
- Run: `GOCACHE="$PWD/.enjoy-logs/go-cache" go test ./internal/bench -run '^TestClaude' -count=1` passed (7.769s). Before-fix failures are retained in claude-structured-regression-red.log, claude-structured-tool-red.log and claude-structured-case-red.log. The original first-pilot artifact loads, compares and re-scores with its failures unchanged; SHA256 remains 6fd3438dd69e682a8c75fd03088b6442a3f0fa0fa85e68f6b9275e237e4f8465.
- Mutation: detection for removing the schema flag and adding text fallback, with baseline/after passes and unchanged source hashes. Evidence under ignored .enjoy-logs/claude-structured-mutation.log and claude-schema-execution-overlay.json.
- Not covered: corrected live execution and authoritative billing. First pilot retained 11 adapter errors (23,075 known tokens, $0.288833 CLI estimate with unknown pricing basis) and one interrupted attempt with unknown usage; these are not successful quality measurements.

Independent read-only correction review found no code blockers. It required clarifying that MAX_STRUCTURED_OUTPUT_RETRIES=1 means one structured-output attempt, and that the native StructuredOutput return tool accompanies Read/Glob/Grep; documentation and recorded budget policy now say so. Clean-tree full verification is required before the corrected live retry; its tested commit and outcome will be recorded in PR #13.


## Completed live Haiku calibration

Full clean-tree `GOCACHE="$PWD/.enjoy-logs/go-cache" task verify` passed with exit 0 at `4d473a424d3971f9c9b1ccb21b241041eb2c01a4`; log `.enjoy-logs/haiku-schema-verify.log`. Doctor parity passed (177 schema tokens); the 6,008 µs single hook observation is above the advertised 5 ms target, not a latency guarantee. Isolated remote-policy preflight passed with no managed settings configured for the organization.

The corrected pilot completed 30/30 passing attempts with 15 evaluable pairs, complete primary usage and 30 distinct native session IDs. All reported model identities matched claude-haiku-5-5; requested low effort remains unverified by native telemetry. Offline rescore preserved every grade, identical comparison JSON and the original SHA256. See `benchmarks/haiku-calibration-pilot-2026-10-19.md` for exact configuration, timing/token variation, hashes, cumulative consumption and billing limitations. No API/test concurrency occurred during the pilot.

The earlier verification block's live-call exclusions describe its historical checkpoint. Current limits are unknown pricing basis/authoritative billing, unknown usage from the first launch's interruption, unreported resolved effort, small task coverage and no equivalence/harness-improvement claim. Codex Luna remains a separately sequenced task.

A sanitized captured-success fixture now locks the actual native StructuredOutput envelope and accounting (954 normalized input, 1,766 output including 1,162 thinking). It omits UUID/timing/permissions and replaces the session ID; provenance is in `internal/bench/testdata/README.md`. The focused recorded-fixture test passed. This fixture adds real CLI evidence alongside the synthetic malformed/failure cases; it does not prove billing.

Independent read-only delivery review approved the report and captured fixture after recomputing the live hash, totals, per-task observations and replay equality, and checking retained fields against the original with the session-ID substitution. The recorded fixture supersedes the earlier historical synthetic-only limitation. Final evidence-checkpoint clean-tree verification is recorded by exact commit and log in PR #13.
