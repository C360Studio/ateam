# Change Proposal: `review-record-130`

## Why

ateam's generated protocol, reviewer contract and merge-check rule row tell every agent to post a `reviewed-by:` PR comment as the review record. AIP repositories forbid a second review record: only their reviewer agent writes `review-implementation.md` (REVIEW-002 format), a person approves it into `approval.md`, and branches merge `--ff-only` with no PR (#130). The owner ruled on the shape on 2026-10-26 (https://github.com/C360Studio/ateam/issues/130#issuecomment-6416383301).

## What Changes

- `.agents/review-policy.json` takes an optional `"review_record": "pr-comment" | "aip"`; absent means `pr-comment`; any other value is rejected by policy validation (`internal/review/policy.go`).
- Under `aip`, `ateam init`, `ateam review setup` and `ateam doctor` use review-record text in `.agents/protocol.md` step 4, the merge-check rule row of `AGENTS.md` and `.agents/rules-index.md`, and the reviewer contract § Record: the record is the archived `review-implementation.md` with `Verdict: PASS` and a matching `approval.md`, ateam agents never post a `reviewed-by:` comment, and merge-check does not apply. The default output is byte-identical.
- `ateam review compare --reviewer-record <path>` reads the reviewer run from the record's Reviewer field (`<agent> (<model>, <effort>)`) and applies the existing tier-and-band comparison; a non-PASS verdict or a missing or non-matching `approval.md` is a finding.
- `docs/model-review.md`, ADR 0002 (addendum) and the model-review spec describe the mode.

## Non-goals

- merge-check reading file records (ruled out).
- Validating the rest of REVIEW-002 (manifest digests against the tree, the findings table): AIP's own gate owns it.
- This repository's own `.agents/review-policy.json` value (stays `pr-comment` by absence).
- The `review-pr` skill text under `aip`: doctor requires its `reviewed-by:` and `Reviewed at` markers (`requiredSkillMarkers` in `internal/bootstrap/bootstrap.go`, which PR #173 rewrites), so a variant is a follow-up.

## REVIEW-002 assumption

Read from AIP's adversarial-review spec (REVIEW-002) and an archived record: a `## Verdict` section holding one-line list items `- **<Name>**: <value>` for Stage, Verdict (`PASS`/`FAIL`), Reviewer, Date and Round; `approval.md` is a table `| Stage | Approver | Date | Review digest |` whose digest is the SHA-256 of the record. AIP's records today write the Reviewer as `<agent-name> (<model-id>, agent type <name>)`, with no effort and an agent name that is not `claude`, `codex` or `gemini`; compare reports that as a finding until the Reviewer value starts with `<agent> (<model>, <effort>)`.
