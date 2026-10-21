# Change Proposal: `review-record-grammar-98`

## Why

Since #75, `scripts/merge-check.sh` accepts a cross-agent review only as a PR comment with `reviewed-by: <agent>` (or a `### Independent Cross-Agent Review` heading) naming exactly one PR commit, written for example `Reviewed at <sha>`. That grammar appeared only in the `merge:check+review` rule rows. The reviewer contract and the protocol did not state it, so a reviewer following them wrote a record that failed the gate (#98, PR #88).

## What Changes

- The reviewer contract (`reviewerContract` in `internal/bootstrap/review.go`, rendered to `.agents/contracts/reviewer.md`) gains a `## Record` section: the two accepted shapes, the single-commit rule, a copyable template, and the re-review rule for later commits that are neither a clean merge of `main` nor docs-only.
- `.agents/protocol.md` Gate 3 and the bootstrap protocol template state the same grammar and point to the contract.
- `AGENTS.md` and `.agents/rules-index.md` rows are unchanged.
