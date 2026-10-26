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
	if record == review.RecordAIP {
		files["AGENTS.md"] = strings.Replace(files["AGENTS.md"], agentsMergeCheckRow, agentsAIPRecordRow, 1)
		files[".agents/rules-index.md"] = strings.Replace(files[".agents/rules-index.md"], indexMergeCheckRow, indexAIPRecordRow, 1)
		files[".agents/protocol.md"] = strings.Replace(files[".agents/protocol.md"], protocolVerifyStep, protocolVerifyStepAIP, 1)
		files[".agents/contracts/reviewer.md"] = reviewerContractFor(record)
	}
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
