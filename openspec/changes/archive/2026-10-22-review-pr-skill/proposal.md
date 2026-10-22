# Change Proposal: `review-pr-skill`

## Why

Every cross-agent review is triggered by the owner opening the other CLI and typing a prompt by hand. The reviewer contract says what a record must contain and `scripts/merge-check.sh` says what it accepts, but no skill walks a reviewer from "a PR was pushed" to "the record comment is posted and the review checkout is gone" (#113). The loop host that would trigger reviews automatically stays outside ateam by design (forge events, vendor apps, or herdr); the harness owes the manual trigger.

## What Changes

- A canonical `review-pr` skill (`.agents/skills/review-pr/SKILL.md`) with thin Claude and Gemini adapters. Codex reads the canonical file. Steps: pick the PR (named, or the one open non-draft PR this agent did not implement and that has no record at its head; ask when zero or several), state the problem and your own approach before opening the diff, review the complete diff and callers on a detached `.worktrees/<agent>-review-pr-<n>` checkout, compare with your approach, run `task verify` on a code PR, post one record comment in the contract grammar, remove the checkout, end with the Ponytail skipped/risk lines.
- `ateam init` ships it: `review-pr` joins `canonicalSkills` and `canonicalSkillBodies` in `internal/bootstrap`, with doctor markers `reviewed-by:` and `Reviewed at`. The template body is the repository's own skill file, asserted by test.
- `.agents/README.md` lists the skill. The harness spec gains a `Shipped review-pr skill` scenario.
- Not changed: merge-check, the reviewer contract, hooks, MCP. No poll, daemon, or dispatcher.
