# Model routing and independent review

> **Status in this repository (ADR 0001, [`docs/adr/0001-review-gate-scope.md`](adr/0001-review-gate-scope.md)).** The merge gate that runs is `scripts/merge-check.sh` (CI job `Merge check`), as described in `AGENTS.md`. It requires `implemented-by:` and `reviewed-by:` lines, each naming an agent with its model and native effort in parentheses (`claude (opus, high)`), and a PR comment review record bound to one PR commit (see `.agents/contracts/reviewer.md` § Record). It passes the runs to `ateam review compare`, which fails only when the reviewer is weaker than an implementer by model tier, then effort band, in `.agents/review-policy.json` (ADR 0002, [`docs/adr/0002-model-tiers-and-effort-bands.md`](adr/0002-model-tiers-and-effort-bands.md); #82). A model or effort the policy doesn't map passes with an `unranked` note. The two lines must name different agents only when the policy sets `cross_provider: true`; it is `false` here, so a same-agent review rests on the reviewer's written record of independence.
>
> The system this document describes is **built and tested but switched off here**:
> - **Policy:** `.agents/review-policy.json` declares model tiers and effort bands only (ADR 0002, [`docs/adr/0002-model-tiers-and-effort-bands.md`](adr/0002-model-tiers-and-effort-bands.md), #81). It configures no routes, so `ateam review route` and `review check` fail closed, setup installs no route cards, and the dashboard reports `unconfigured`.
> - **JSON review records, reviewer-executed required checks and owner exceptions** are off. `ateam review check` is invoked by neither CI nor merge-check. ADR 0001 lists the triggers for enabling them.
>
> Where this document says "approval blocks", it means `ateam review check`, not the merge gate.

The rest of this document describes the dormant path; the merge gate's tier-and-band rule above does not apply to it. ATeam validates project-approved model relationships and commit-bound review evidence. The reviewer must be a separate run that wrote none of the implementation and use a model approved as stronger than **every actual implementation model**. Same-provider review is allowed unless the project also requires provider separation. Reasoning effort is recorded separately; it never increases a model's rank.

If development already uses the highest approved model, approval blocks until an explicitly authorized owner grants a narrow exception. The exception waives superiority only. Independence, provider separation, verification, a passing verdict and fresh evidence still apply.

## Configure an approved policy

No model ranking is bundled or inferred from names, prices or benchmark scores. Existing repositories without `.agents/review-policy.json` keep working with `init` and `doctor`; review commands fail closed until configured. The owner must approve exact model identities and stronger-than relationships before use.

This is a schema illustration with **placeholder IDs, not a runnable model recommendation**. Replace the model identities, owner/reviewer forge logins and check names with approved project values:

```json
{
  "version": 1,
  "models": [
    {"id": "APPROVED-DEVELOPER-ID", "provider": "codex"},
    {"id": "APPROVED-REVIEWER-ID", "provider": "claude", "stronger_than": ["APPROVED-DEVELOPER-ID"]}
  ],
  "routes": {
    "developer": {"model": "APPROVED-DEVELOPER-ID", "effort": "medium"},
    "reviewer": {"model": "APPROVED-REVIEWER-ID", "effort": "high"}
  },
  "owners": ["project-owner"],
  "review_authors": ["review-bot"],
  "required_checks": ["task verify"],
  "cross_provider": false
}
```

`stronger_than` comparisons are transitive and must be acyclic. Unknown IDs, unresolved inheritance, contradictory comparisons and malformed policies are rejected. Setup supports the `claude`, `codex` and `gemini` provider identities. They identify the client/provider route, not a cross-provider ranking.

```sh
ateam review setup --policy approved-policy.json
ateam doctor
ateam review route
```

Setup validates the policy structure and creates `.agents/review-policy.json`, canonical contracts and `.agents/routes/developer.md` / `reviewer.md` launch cards. Claude routes get a thin skill pointer; Gemini CLI routes get a custom-command pointer; Codex reads the canonical card directly. Gemini CLI is not Antigravity. There is no fabricated universal native-agent configuration.

Setup checks all destinations and reserved pointers for non-selected providers before writing, preserves existing custom configurations and refuses symbolic-link destinations. Identical setup is idempotent. Local contract additions are allowed when canonical obligations remain intact. Unexpected I/O failures may leave the reported subset of new files; reconcile those files and retry. Setup never overwrites conflicts, even during a policy change: inspect and explicitly reconcile tracked policy/cards/pointers as a reviewed change. Doctor always checks the canonical role contracts, rejects orphan route cards/pointers without a policy, and checks configured policy validity and instruction drift; it does not certify runtime model selection or review approval. A structurally valid ceiling policy can be installed even though its route is ineligible without an exception.

`review route` checks model intent before work. After escalation, supply **all** actual implementation model IDs:

```sh
ateam review route --developer-models approved-dev-id,approved-escalated-id
```

The operator starts the native client with the indicated exact model and effort. Cards and pointers do not launch agents, pin a running client's model, enforce a sandbox or prove its identity. Report an unavailable route and stop; do not silently downgrade. Native runtime pinning and cryptographic model attestation are not implemented.

## Upgrade and troubleshooting

After installing a newer ATeam binary, run `ateam init` without `--force` and inspect its diff, then run `ateam doctor --check`. Initialization adds missing canonical contracts and preserves existing files; it does not migrate custom instructions or enable model review. Do not use `--force` as a general repair command: it overwrites generated files.

| Symptom | Resolution |
| --- | --- |
| Missing or drifted canonical contract, even without policy | Use `ateam init` to add missing contracts. For existing customized contracts, reconcile the canonical obligations explicitly while retaining project additions; doctor does not repair them. |
| Unexpected reserved provider pointer | Inspect the reported `.claude/skills/ateam-{developer,reviewer}/SKILL.md` or `.gemini/commands/ateam-{developer,reviewer}.toml`. Reconcile stale pointers from the previous provider as a reviewed change, preserving any custom content elsewhere before removing an obsolete pointer. Setup refuses existing inactive pointers, including dangling links, before writing. |
| Orphan route card or pointer after policy removal | Decide whether to restore the approved policy or retire the obsolete instructions in the same reviewed change. Doctor checks these paths even without policy. |
| Setup conflicts while changing policy | Review the policy, role cards, and selected provider pointers together. Setup preserves existing conflicting files; explicitly reconcile them before retrying. |
| Route rejects a ceiling model | Do not downgrade the developer silently. Arrange the explicit owner exception described below; `route` itself has no exception flag, and the commit-bound waiver is evaluated by `review check`. |
| Check reports missing baseline policy or stale scope | Land initial policy separately, ensure the checkout contains the current default-branch base, and use fresh evidence for the current head and governing policy. Candidate policy cannot grant its own approval. |

Doctor checks files and parity, not approval or runtime identity. A linked provider directory without a reserved role pointer is not itself an inactive-pointer conflict; selected generated destinations still refuse symbolic links.

## Review and record evidence

Follow [the developer contract](../.agents/contracts/developer.md) and [the independent reviewer contract](../.agents/contracts/reviewer.md). The reviewer examines the complete diff and affected callers, tries to refute findings, and reports concrete mechanisms and corrections. Only the author changes the deliverable. Use native read-only controls when available and disclose any enforcement gap.

A record contains version, repository identity, full base/head commits, the SHA-256 digest of the exact governing policy bytes, every implementer run, the reviewer run, read-only attestation, verdict, findings, checks and limitations. Runs contain `id`, `model`, `provider` and `effort`. Each check names its `commit`, `status` (`passed`, `failed`, `unavailable` or `skipped`) and `source` (`reviewer` or `implementer`) so reviewer-run evidence remains distinct from implementer reports. Required checks need reviewer-executed passing evidence at the current head; an implementer report alone does not satisfy them. Unavailable evidence is not a pass. `approve` requires no unresolved `blocking` or `high` findings and all configured verification.

A schematic record follows; replace every placeholder with the actual full commit/digest and run metadata:

```json
{
  "version": 1,
  "repository": "github:https://api.github.com/owner/repo",
  "base": "FULL_BASE_COMMIT_SHA",
  "head": "FULL_REVIEWED_HEAD_SHA",
  "policy_digest": "SHA256_OF_EXACT_BASELINE_POLICY_BYTES",
  "implementers": [{"id": "developer-run-1", "model": "APPROVED-DEVELOPER-ID", "provider": "codex", "effort": "medium"}],
  "reviewer": {"id": "independent-review-run-1", "model": "APPROVED-REVIEWER-ID", "provider": "claude", "effort": "high"},
  "read_only": true,
  "verdict": "approve",
  "findings": [],
  "checks": [{"name": "task verify", "commit": "FULL_REVIEWED_HEAD_SHA", "status": "passed", "source": "reviewer"}],
  "limitations": ["Runtime model identity and implementation completeness are attestations."]
}
```

An authorized reviewer posts a PR issue comment whose exact first line is `ateam-review`, followed by the record JSON. If superiority needs a waiver, a login in the trusted policy's `owners` posts a separate comment beginning `ateam-review-exception`, followed by an exception JSON object with `version`, `repository`, `head`, `reviewer_model`, `policy_digest` and nonempty `reason`. An exception is specific to that head, reviewer model and policy revision; it is not a permanent exemption.

For example, the owner posts the following comment body after replacing every placeholder with the reviewed values. The first line must be the marker shown; do not include the surrounding Markdown code fence in the posted comment. Owner identity comes from the authenticated comment author, not a JSON field.

```text
ateam-review-exception
{
  "version": 1,
  "repository": "github:https://api.github.com/owner/repo",
  "head": "FULL_REVIEWED_HEAD_SHA",
  "reviewer_model": "APPROVED-REVIEWER-ID",
  "policy_digest": "SHA256_OF_EXACT_BASELINE_POLICY_BYTES",
  "reason": "Implementation used the highest approved model; I authorize this independent reviewer for this commit only."
}
```

This waiver does not replace the review record or waive independence, configured provider separation, required checks, or a passing verdict. New commits require fresh scoped evidence.

```sh
ateam review check --pr 15 --record-comment 12345
ateam review check --pr 15 --record-comment 12345 --exception-comment 12346
task review:check -- --pr 15 --record-comment 12345
```

These commands fetch evidence; they do not post comments or grant approvals on the forge. The checker requires the clean local HEAD to match the live PR head and requires the PR to target the repository default branch, verifies its repository/ref and live branch SHA through forge metadata, and reads policy from that verified commit. Initial policy adoption therefore needs a separate owner-authorized configuration PR merged into the baseline; a candidate branch cannot approve its own policy or add itself as an exception owner. The checker authenticates the comment author against `review_authors` (review) or `owners` (exception), binds evidence to the repository/base/head/policy and rejects missing edit metadata or known edited comments, rereads the exact comments, and rereads PR revisions to detect movement. Post fresh comments instead of editing evidence. GitHub and Gitea inputs have the same validation requirements. Missing, partial or unavailable evidence blocks approval.

Repository identities use `github:https://api.github.com/owner/repo` or `gitea:https://host/owner/repo`. Forge timestamps may have only second-level precision: equal creation/update timestamps cannot prove that no same-tick edit occurred before validation. Original comment authorship is authenticated, but current body authorship is not proven against every malicious writer. A shared forge login and self-reported model metadata remain attestations. They cannot prove who controlled the original model run, that every implementer was disclosed, that the reviewer never wrote implementation, or that a shared owner login represents a separate human. State these limits in the review. A successful eligibility result alone is never substantive code approval.

Every changed head or governing policy invalidates old `ateam review check` evidence and exceptions. Limitation: `validateBinding` (`internal/review/record.go`) binds to the exact head with no exemption. Under a ruleset that requires an up-to-date branch, every PR therefore needs a fresh record and a fresh reviewer-executed check after its merge-of-`main` commit. The intended rule, which merge-check already enforces for its own records (#75), keeps a record valid across a merge of `main` that touches none of the PR's files and across documents-only commits. `validateBinding` does not implement that exemption yet; implement it before enabling `ateam review check` as a gate. `ateam review check` has no docs-only relaxation or filename classifier. The merge gate that runs does: `scripts/merge-check.sh` treats a PR as documents only when every changed path ends in `.md`, or is under `openspec/` and ends in `.yml`, `.yaml` or `.json`. Paths under `.claude/agents/` and `.agents/contracts/` are always code. A documents-only PR needs no review lines. Review instructions, policy, executable Markdown and both sides of renames still receive substantive review; a `.md` suffix does not exempt them from the reviewer contract. Hooks do no network/model work and the existing two-tool MCP interface remains unchanged.

Review evidence uses trimmed `GITEA_TOKEN`, falling back to trimmed `ATEAM_GITEA_TOKEN`. GitHub uses trimmed `GH_TOKEN`, then `GITHUB_TOKEN`; whitespace-only values do not suppress the fallback.

## Dashboard routing display

The PM dashboard shows the selected repository's configured developer and reviewer model IDs, providers and reasoning efforts. `GET /api/model-routing` returns the same `model_routing` summary included in `GET /api/state`; both use existing read authentication and `Cache-Control: no-store`. Refresh the page to re-read policy.

Statuses are `eligible` (configured routes pass policy eligibility), `blocked` (valid policy with ineligible routes, including a reason), `unconfigured` (missing policy), `invalid` (malformed or over 1 MiB), and `unavailable` (read failure or unsafe/non-regular file). Eligible and blocked summaries include `developer`, `reviewer` and `policy_digest`; each route has `model`, `provider` and `effort`. Other states omit route metadata. The read is confined to the selected repository and does not expose policy owners or review authors.

This is a read-only view of local configuration, not trusted default-branch PR approval evidence or live model identity. It does not launch agents, edit policy, apply owner exceptions, or verify generated route-card drift. Effort remains separate from model strength and Ponytail mode.
