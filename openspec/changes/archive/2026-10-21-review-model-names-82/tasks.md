## 1. Implementation

- [x] 1.1 Require a parenthesised model on the `implemented-by:` and `reviewed-by:` lines in `scripts/merge-check.sh` (#82).
- [x] 1.2 Add `TestMergeCheckReview/model_named_on_both_lines` (missing, blank, and comma-separated models) and record `cp` + SHA-256 mutation detection for both line checks and the blank-model pattern (#82).
- [x] 1.3 Update the rule rows, bootstrap templates, and harness spec scenario (#82).
- [x] 1.4 Bind the model to the parenthesised group directly after the agent name, from the Codex review of `4849ced`: `(author) claude` and `claude () (note)` no longer pass (#82).
