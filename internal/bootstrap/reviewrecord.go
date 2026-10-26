package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c360studio/ateam/internal/review"
)

// The review-record text for a repository whose policy sets review_record: aip
// (owner ruling on #130). Each replaces one default constant, so the default
// output is untouched.
const (
	agentsAIPRecordRow = `| ` + "`review`" + ` | ` + aipRecordRule + ` | ` + "`.agents/review-policy.json`" + `; ` + "`.agents/protocol.md`" + ` |`
	indexAIPRecordRow  = `| ` + "`review`" + ` | ` + aipRecordRule + ` | ` + "`ateam review compare --reviewer-record`" + ` |`
	aipRecordRule      = `Under ` + "`review_record: aip`" + ` the review record is the archived ` + "`openspec/changes/archive/<date>-<id>/review-implementation.md`" + ` with ` + "`Verdict: PASS`" + ` and a matching ` + "`approval.md`" + `, written only by the project's reviewer and approved by a person; ateam agents never post a ` + "`reviewed-by:`" + ` PR comment; the record's Reviewer (` + "`<agent> (<model>, <effort>)`" + `) is not weaker than every implementer by tier then band (` + "`ateam review compare --reviewer-record`" + `); ` + "`task merge:check`" + ` does not apply, since branches merge ` + "`--ff-only`" + ` without a PR`

	protocolVerifyStepAIP = `4. **Verify (4-Gate Definition of Done):** ` + "`task verify`" + ` green, ` + "`ateam doctor`" + ` parity green, ` + "`ateam bench --selftest`" + ` green, a not-weaker reviewer recorded in the archived ` + "`openspec/changes/archive/<date>-<id>/review-implementation.md`" + ` with ` + "`Verdict: PASS`" + ` and a matching ` + "`approval.md`" + ` (` + "`review_record: aip`" + ` in ` + "`.agents/review-policy.json`" + `; check it with ` + "`ateam review compare --implementer <agent:model:effort> --reviewer-record <path>`" + `; template: ` + "`.agents/contracts/reviewer.md`" + `), and Ponytail exit summary stated. ateam agents never post a ` + "`reviewed-by:`" + ` PR comment, and ` + "`task merge:check`" + ` does not apply: the branch merges ` + "`--ff-only`" + ` without a PR.`

	closeRuleRowAIP = `| ` + "`review`" + ` | Under ` + "`review_record: aip`" + ` there is no PR: a branch archives its completed OpenSpec change and lands by the project's own ` + "`--ff-only`" + ` merge into ` + "`main`" + `, never a squash or auto-merge; once it has landed, the claiming agent removes its worktree and branch | ` + "`.agents/protocol.md`" + ` § Close |`

	protocolClaimStateAIP = `- **Who has claimed what:** An agent-prefixed branch (` + "`claude/<topic>`" + `, ` + "`codex/<topic>`" + `, ` + "`gemini/<topic>`" + `) in its own worktree (` + "`git worktree list`" + `). Under ` + "`review_record: aip`" + ` no PR backs the claim; the project's own workflow records the claim.`

	protocolClaimStepAIP = `2. **Claim:** Create or switch to an isolated worktree on an agent-prefixed branch (` + "`claude/…`" + `, ` + "`codex/…`" + `, ` + "`gemini/…`" + `). Under ` + "`review_record: aip`" + ` open no PR: follow the project's own workflow for the claim, the review and the approval.`

	protocolCloseStepAIP = `5. **Handoff or Land (Close):** At a session boundary or before context compaction, run ` + "`ateam state handoff`" + ` (or ` + "`/handoff`" + `) to record the 5-field pickup prompt. Under ` + "`review_record: aip`" + ` the branch lands by the project's own ` + "`--ff-only`" + ` merge into ` + "`main`" + ` with no PR, squash or auto-merge, after the OpenSpec change is archived (` + "`openspec/changes/<id>/`" + ` → ` + "`openspec/changes/archive/<date>-<id>/`" + `) with its approved review record, as the project's workflow orders it; the change's ` + "`Hold: OPEN-Q proposal awaits owner approval`" + ` must be gone before the archive move. Once the branch has landed, remove the worktree you claimed and delete its branch. Pickup labels every finished worktree; one older than a day is yours to explain or remove.`

	reviewPRSkillAIP = `---
name: review-pr
description: >-
  Under review_record aip, check the reviewer strength of a REVIEW-002 review record.
  ateam agents write no review record and post no reviewed-by comment.
---

# Check a Review Record (review_record: aip)

Read [the shared protocol](../../protocol.md) and [the reviewer contract](../../contracts/reviewer.md) first. This repository's policy sets ` + "`review_record: aip`" + `: the project's own reviewer writes the review record and a person approves it. You never write or change a review record, and you never post a ` + "`reviewed-by:`" + ` comment.

## 1. Find the Record

Use the change the user named. The record is ` + "`openspec/changes/archive/<date>-<id>/review-implementation.md`" + ` with ` + "`approval.md`" + ` beside it. With no record or no approval, say so and stop; do not review in its place.

## 2. Check It

Run ` + "`ateam review compare --implementer <agent:model:effort>... --reviewer-record <path>`" + ` naming every implementing run, and report its output verbatim. Exit 0: the record is an approved PASS from a reviewer not weaker than every implementer. Exit 1: it lists the findings. End your reply with 1–2 lines naming what you skipped or did not check and any residual risk.
`

	reviewerRecordAIP = `## Record

This repository's policy sets ` + "`review_record: aip`" + `. The review record is the archived ` + "`openspec/changes/archive/<date>-<id>/review-implementation.md`" + ` in the REVIEW-002 format, with ` + "`Verdict: PASS`" + `, and a matching ` + "`approval.md`" + ` beside it whose implementation row holds that record's SHA-256. Only the project's own reviewer writes that record and only a person approves it. ateam agents never write or change it and never post a ` + "`reviewed-by:`" + ` PR comment.

The record's Reviewer field starts with ` + "`<agent> (<model>, <effort>)`" + ` (for example ` + "`claude (opus, high)`" + `) so that ` + "`ateam review compare --implementer <agent:model:effort>... --reviewer-record <path>`" + ` can check it is not weaker than every implementer; that command also rejects a verdict other than PASS and a missing or non-matching ` + "`approval.md`" + `. ` + "`task merge:check`" + ` does not apply: the branch merges ` + "`--ff-only`" + ` without a pull request.
`
)

// reviewerContractFor returns the reviewer contract for a review_record value.
func reviewerContractFor(record string) string {
	if record == review.RecordAIP {
		return reviewerContractBody + reviewerRecordAIP
	}
	return reviewerContract
}

// templateFiles returns the init templates for root: the defaults, with the
// review-record text swapped when root's review policy sets review_record: aip.
func templateFiles(root string) (map[string]string, error) {
	files := defaultTemplateFiles()
	record, err := policyReviewRecord(root)
	if err != nil {
		return nil, err
	}
	if record != review.RecordAIP {
		return files, nil
	}
	for rel, swaps := range map[string][]string{
		"AGENTS.md":              {agentsMergeCheckRow, agentsAIPRecordRow, closeRuleRow, closeRuleRowAIP},
		".agents/rules-index.md": {indexMergeCheckRow, indexAIPRecordRow, closeRuleRow, closeRuleRowAIP},
		".agents/protocol.md": {protocolClaimState, protocolClaimStateAIP, protocolClaimStep, protocolClaimStepAIP,
			protocolVerifyStep, protocolVerifyStepAIP, protocolCloseStep, protocolCloseStepAIP},
	} {
		files[rel] = strings.NewReplacer(swaps...).Replace(files[rel])
	}
	files[".agents/contracts/reviewer.md"] = reviewerContractFor(record)
	files[".agents/skills/review-pr/SKILL.md"] = reviewPRSkillAIP
	return files, nil
}

// policyReviewRecord reads review_record from root's policy; no policy is the default.
// An unreadable or invalid policy fails init rather than guessing the record mode.
func policyReviewRecord(root string) (string, error) {
	if err := rejectReviewSymlinks(root, reviewPolicyPath); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, reviewPolicyPath))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", reviewPolicyPath, err)
	}
	policy, err := review.ParsePolicy(data)
	if err != nil {
		return "", fmt.Errorf("%s: %w", reviewPolicyPath, err)
	}
	return policy.ReviewRecord, nil
}
