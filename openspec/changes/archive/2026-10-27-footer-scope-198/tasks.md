## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#198). Evidence: the owner approved in Enjoy on 2026-10-27, and Claude wrote the header at their request.

## 2. Implementation

- [x] 2.1 Rescope the rule in `AGENTS.md:13,64`, `.agents/rules-index.md:15`, and `.agents/protocol.md:43` Gate 4 (#198). Evidence: the two rule rows carry the same text; `ateam doctor --check` passes with no violations.
- [x] 2.2 Rescope `.agents/skills/ponytail/SKILL.md:5,11`, `.agents/skills/preflight/SKILL.md:31`, and `.agents/skills/review-pr/SKILL.md:35` (#198). Evidence: ponytail names each task conclusion and handoff and exempts acknowledgements and progress updates; preflight and review-pr say "End your conclusion".
- [x] 2.3 Mirror 2.1–2.2 into `internal/bootstrap/templates.go:183,204,245,436,442,510` and `internal/bootstrap/reviewrecord.go:47` (#198). Evidence: a scratch `ateam init` writes the new wording into `AGENTS.md`, `rules-index.md`, and the skills. The generated review-pr skill is byte-identical to the canonical one. The generated ponytail and preflight skills were already shorter than the canonical files before this change; only the footer sentence was changed in each. `templates.go:474` already says "ending with Ponytail's ... summary" and needed no change. The init `protocol.md` has no Gate 4 line.
- [x] 2.4 Change the last line of `LadderPrompt` in `internal/hook/hook.go:430` to "End conclusions and handoffs with 1-2 lines" (#198). Evidence: `go test ./internal/hook/ ./internal/bootstrap/ ./cmd/...` green; doctor hook latency 8748µs against the 10000µs budget.
- [x] 2.5 Verify (#198). Evidence: the grep for "end every reply", "End reply with", "Agent replies end", "End your reply", "End your response", and "Reply concludes" outside `benchmarks/`, `archive/`, and this change finds nothing; `task verify` green on a clean tree (see PR).
