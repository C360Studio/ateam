# ADR 0001: Scope of the model review gate

- **Status:** Accepted (owner, 2026-10-21)
- **Date:** 2026-10-21
- **Deciders:** repository owner
- **Evidence:** Fable dogfood sessions 2026-10-19 to 2026-10-21, Enjoy conversation "Agent review of ateam dogfooding"

## Context

Two review systems exist in this repository.

**Merge-check** (`scripts/merge-check.sh`, CI job `Merge check`, required by the `main` ruleset) is live. It reads two PR body lines, `implemented-by:` and `reviewed-by:`, requires them to name different agents among `claude`, `codex`, `gemini`, exempts drafts and documents-only PRs (classification by extension since #66), and blocks on open `class:flake` issues. CI runs `task verify` at every head. Eight code PRs merged under it on 2026-10-20 (#57, #65, #68, #69, #70, #71, #78, #79), each with a prose review-record comment.

**Model review** (`docs/model-review.md`, `internal/review`, `internal/bootstrap/review.go`, `internal/web/routing.go`) is built and tested but switched off. It adds an owner-declared model ranking with transitive, acyclic `stronger_than` relations; JSON review records bound to repository, base, head and policy digest; reviewer-executed required checks at the exact head; owner exception comments for ceiling models; and comment edit detection. The package has 23 tests, setup and doctor drift checks have 11, and the dashboard endpoint exists. In this repository there is no `.agents/review-policy.json`, `ateam review route` reports unconfigured, zero JSON records have ever been posted, and `ateam review check` is invoked by neither CI nor merge-check.

The document reads as a description of the operating gate. Two of its claims are false for this repository as it runs:

1. "The gate applies to every PR without a docs-only relaxation or filename classifier." The gate that runs has that classifier.
2. "Every changed head or governing policy invalidates old evidence." `validateBinding` (`internal/review/record.go:174`) binds the record to the exact head with no exemption. The `main` ruleset requires an up-to-date branch, so every PR receives a merge-of-main commit before it can merge. Enabling the record check as built would therefore require a fresh record and a fresh reviewer-executed `task verify` after that commit, on every PR.

Three further facts bound what the built system can guarantee here:

- Every agent posts to GitHub as the same login. `review_authors` and comment-author authentication cannot distinguish Claude from Codex from Gemini, so independence is an attestation under either system. The document says this in its limitations paragraph.
- All three providers in use run at their model ceilings. The policy blocks a ceiling developer until an owner posts an exception comment, so the exception path would be the common path.
- The one guarantee merge-check cannot provide and model review can is reviewer strength: a reviewer model at least as strong as every implementer model. The global rule "a reviewer is never weaker than the implementer" is currently enforced nowhere; on the eight PRs above the reviewer model was named on four.

## Decision

Keep both systems. Enable model review in stages, through the gate that already runs, and record the triggers for the rest.

1. **Declare the ranking.** The owner writes `.agents/review-policy.json` naming the models in use and the `stronger_than` relations they will stand behind, runs `ateam review setup`, and commits the generated contracts and cards. Per-PR cost: zero. (#81)
2. **Enforce strength in merge-check.** Both body lines must name the model in parentheses. When a policy exists, merge-check compares reviewer against every implementer model through a small `ateam review compare` subcommand that reuses `internal/review.Policy`; without a policy the comparison is reported as skipped. Per-PR cost: one word per line. (#82)
3. **Bind evidence to commits the semengine way.** A review record is a PR comment naming the commit it read; a later content commit invalidates it; a merge of `main` touching none of the PR's files, and docs-only commits, keep it. This is #75 and gives the head-binding guarantee without the re-review loop.
4. **Make the document true.** A status block states which parts are enabled, the classifier sentence describes merge-check, and the head-binding rule is amended with the merge-of-main exemption, implemented in `validateBinding` or marked as a limitation until it is. (#83)
5. **Leave JSON records, reviewer-executed checks and exceptions off by default.** The code stays; `ateam review check` remains available for repositories that want it.

Steps 1, 2 and 4 are independent of each other except that step 2's strength half needs step 1's policy. Step 3 is tracked separately in #75 and should land before or with step 2, since both edit the same block of merge-check.

### Triggers for enabling the rest

Switch on JSON records, reviewer-executed checks and owner exceptions when any one of these holds:

- Agents acquire distinct forge identities, so independence stops being an attestation and `review_authors` can be enforced.
- A second repository adopts the harness and needs machine-readable review records across repositories.
- A review failure occurs on this repository that steps 2 and 3 would not have caught.

Until then, the heavy path is not a gate here and the document must not say otherwise.

## Consequences

**Positive.** Reviewer strength becomes enforced on every code PR for the cost of a model name. Evidence binding arrives through #75 without the per-PR re-review tax. The policy file gives `review route`, doctor drift checks and the dashboard real content. No code is deleted and no new schema is introduced. The document stops describing a gate that does not run.

**Negative.** Independence stays an attestation until forge identities exist. Ceiling routes will show as blocked in `review route` and the dashboard, which is accurate and may confuse a reader who expects green; the status block in step 4 explains it. The JSON record format remains untested against real PRs in this repository, so a future enablement will find friction the tests did not.

**Neutral.** `internal/review` stays in the binary and in `task verify`. The merge-check script grows one jq clause and one subcommand call.

## Alternatives considered

- **Enable the full model-review gate now.** Rejected for this repository: the exact-head binding plus the up-to-date ruleset forces a re-review and reviewer-run verify on every PR; the ceiling policy makes the exception path routine; and the independence guarantee collapses to attestation under a shared login. The gains that survive are strength and binding, which steps 2 and 3 deliver more cheaply.
- **Delete the model-review code and document.** Rejected: the policy lattice and record validation are the right shape for multi-repository or multi-identity use, are tested, and cost nothing while dormant. The adoption document already warns that other repositories may want them.
- **Enforce strength by convention only.** Rejected: the eight PRs show the model name is omitted when nothing requires it.

## References

- `docs/model-review.md`
- `AGENTS.md` rows `verify+review` (configured model review) and `merge:check+review`
- `scripts/merge-check.sh`, `.github/workflows/ci.yml`
- `internal/review/record.go` (`validateBinding`, `validateChecks`), `internal/review/policy.go` (`stronger`)
- Issues #46, #66, #75, #81, #82, #83; semengine `.agents/rules-index.md:109` for the cross-agent review rule being ported
