# Design: Model review policy port

## Baseline and provenance

ATeam baseline: `26c98c29365be118099b40c982fe4b00c5f44c03`. Semengine inspected at `2422d8fb975212a05532dd1940f4d9de24596994`:

- `.agents/README.md`, model routing: distinguishes model tier and effort; allows equal-tier review; does not rank across providers.
- `.agents/protocol.md`, cross-agent review: independence, read-only review, commit scope and evidence attribution.
- `.agents/contracts/semengine-reviewer.md`, workflow and verdict: examine callers, refute findings, give mechanism/fix/verification, and distinguish unavailable evidence.
- `internal/harness/contract/agentsetup_test.go`: adapter parity and sensitivity tests.
- `scripts/merge-check.sh`: GitHub-specific provider-label gate, with explicitly documented attestation/freshness limits. Port its intent into Go, not the whole shell implementation.

ATeam already has stdlib-based bootstrap, task execution and forge boundaries. Reuse those; do not create another orchestrator or model runner.

## Policy decisions

Keep identity, provider, model strength and reasoning effort separate. A distinct reviewer run must have written none of the implementation. Same-provider review is valid by default. An optional cross-provider requirement is an additional condition, not evidence of superiority.

Use a tracked, versioned policy (proposed `.agents/review-policy.json`) with exact model identities and explicit approved stronger-than comparisons. Do not infer ordering from names or pricing. Initial comparisons are project-approved declarations, not benchmark claims; unknown identities and unresolved `inherit` are ineligible. Reject contradictory policy relationships. Validate routing before implementation starts, and revalidate against all actual implementation models at review time, including escalations.

The reviewer must outrank every implementation model. If no eligible model is available, block approval. The owner explicitly chose this behavior at the model ceiling. A narrowly scoped owner exception may waive superiority for the named commit and reviewer, with owner, reason and policy revision recorded. It cannot waive reviewer independence, other configured constraints, required verification, current evidence or the verdict. No permanent exemption is implied. New commits or policy revisions require fresh evidence/exception approval.

## Review and evidence

Start with developer/reviewer roles. Keep the reviewer read-only with respect to the deliverable; use native sandbox controls where supported and honestly report enforcement gaps. The author fixes findings. Evidence records the diff base and reviewed head, policy revision, all implementer run identities/models, reviewer run identity/provider/model/effort, checks actually run, checks only reported by others, unavailable evidence, findings and verdict.

Review the complete diff and affected callers, apply only triggered project rules, try to refute findings, and name concrete failure mechanisms and minimal corrections. Approval requires no blocking/high findings and the required verification evidence. A model-eligibility result alone is not code approval. Initially any changed head invalidates approval; do not port Semengine's merge-only carry-forward optimization yet.

Executable instructions, role adapters/contracts and policy/configuration changes require substantive review even when their suffix is `.md`. Check rename source and destination paths. Any docs-only relaxation must not classify these as ordinary prose.

Use a small Go package for policy and record validation, with a CLI/Taskfile entrypoint usable through the existing `ateam_task` tool. Forge adapters supply equivalent normalized GitHub/Gitea inputs; missing, partial or unavailable remote evidence cannot become approval. A shared forge login and self-reported model metadata establish attestations, not cryptographic proof of the originating model or owner. Verify the available source identity and disclose the remaining trust boundary. Policy edits cannot self-authorize a waiver using only candidate-branch data: resolve the governing policy/owner authority from the trusted baseline or an explicit authorized policy transition.

## Ownership and integration order

The routing/review work owns the new policy package, its tests and this OpenSpec change. At inspection, open PR #12 (`codex/mcp-registration`, issue #11) changes bootstrap templates, doctor/config checks, CLI and shared instructions; merge or reconcile that work before touching the same integration surfaces. PR #13 (`codex/benchmark-calibration`, issue #10) owns benchmark adapters and `cmd/ateam/bench.go`; preserve its changes and keep routing comparisons independent of benchmark scores. The UI agent's uncommitted `internal/state`, `internal/web` and `ui/` work stays outside this claim. Inspect live overlap again before implementation; this inventory is not a lock on others' work.

## Implementation slices and acceptance

1. Policy model, parser and eligibility checks; fixtures for known/unknown/equal/weaker/stronger models, multiple implementers and provider constraints; then bootstrap/adapters and doctor drift checks that preserve custom configuration.
2. Portable review contract and record/exception format; fresh commit/policy binding, independent identity checks, review-sensitive path classification and explicit verdict/evidence requirements.
3. CLI/Taskfile and GitHub/Gitea integration, without hook network/model calls or new MCP tools; complete parity, mutation, generated checks and independent review.

Tests use requirement-owned expected outcomes, not production comparisons as their oracle. Add refusal mutations (e.g. accepting equal-tier review, ignoring a new commit, accepting an unscoped exception). Use generated policy/input combinations and review histories against a small test-owned reference model; prove assertion activation and boundary reach. Record seeds, completed checks and detection/survivor/invalid/inconclusive outcomes. Run required repository gates before implementation completion.

## Implemented decisions and limits

Policy is explicitly installed with `review setup --policy FILE`; no universal ranking is bundled. Portable role cards plus Claude skill/Gemini command pointers describe exact model/effort intent. They do not pin native runtime settings or launch agents; the operator must select the model and disclose actual identities. This replaces the proposed native-adapter pinning with the smallest portable mechanism supported by existing adapter conventions.

The gate reads policy from the live default-branch commit verified through forge repository/branch metadata, not an arbitrary PR target. Initial policy adoption must land separately under owner control. Configured `review_authors` authenticate permitted record-posting accounts independently of exception `owners`. Required check evidence must be reviewer-executed; all PR files require review, eliminating docs/rename bypasses without a separate file classifier. Current review and exception comments are pinned by ID and revalidated before a successful result. Shared-account, runtime identity and implementation-completeness claims remain attestations.

No UI/benchmark implementation, new MCP tool, automatic merge, branch-protection mutation, native model launcher, universal ranking or cryptographic runtime attestation is introduced. PR #12 remains open; integration was kept in separate review files with minimal existing-file hooks, and current main was merged without overwriting its registration changes. Final integration must preserve its pending edits.
