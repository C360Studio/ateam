# Change Proposal: `review-gate-wording-325-322`

Approved-by:

## Why

- **#325 (governance note G1).** `gateIsDoc` keeps `.claude/agents/` and `.agents/contracts/` as code (`internal/review/gate.go:712-716`), but `.agents/skills/review-pr/SKILL.md` is ordinary Markdown. A PR that changes only the reviewer's instructions is documents only and needs no review record.
- **#322 (HAAG E3, finding F5).** The `ateam/review-record` status reads `<VERDICT> by <reviewer> at <sha>` (`internal/state/commitstatus.go:154`) and `Merge check` says `reviewing agent: <agent>` (`internal/review/gate.go:395`). Both names come from lines in the record comment, and no check reads who wrote the comment. The limit is stated in `.agents/protocol.md`, not on the surfaces a reader sees.

## What Changes

- `gateIsDoc` treats any path under `.agents/skills/review-pr/` as code (#325).
- The status description becomes `<VERDICT> self-reported by <reviewer> at <sha7>`. The words come before the reviewer so that the 140-byte cut never drops them (#322).
- The passing merge-check line adds `; the reviewer and models are self-reported, the comment author is not checked` (#322).
- The code-PR definition in `AGENTS.md`, `.agents/rules-index.md`, their `ateam init` templates (`internal/bootstrap/templates.go`), `docs/model-review.md` and the harness spec names `.agents/skills/review-pr/`. `docs/adoption.md` shows the new status text.

## Track

`design`: it edits `internal/review/`, a gate file. See `design.md`.

## Non-goals

- No change to `scripts/gate-files.sh`: the skill stays outside the gate-file pattern, so it needs a review record, not owner approval.
- The thin adapters (`.claude/skills/review-pr/`, `.gemini/commands/review-pr.toml`) stay documents. They only point at the canonical skill, and `ateam doctor` checks that they match it.
- No check of who wrote the record comment. The wording states the limit and does not close it.

## Definition of Done

- `TestGateGitea` covers a PR that changes only `.agents/skills/review-pr/SKILL.md` (code, needs a record) and asserts the self-reported summary. `TestChooseStatusesD3Table` asserts the new status text. `task mutate:check` reports `detection` for each of the three changes.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green. The change is archived before the PR is marked ready. The owner applies `gate-change-approved` and comments at the final head.
