# Proposal: whole-review-marker-399
Approved-by: Coby Leuschke (2026-11-24) sha256:10b7a17c4a6b

## Track

`design`: adds a harness Requirement, a new `ateam review whole` command adopters run, a new pickup-line field, and a protocol Close-step sentence in the generated protocol.

## Decisions you are making by approving

- When someone finishes reading the whole repository as one system, they record it with one command, and that record is shared with everyone who fetches the repository.
- Every session start shows how many commits (one per PR under squash merge) and lines have landed on the default branch since that record, so a growing gap is visible without anyone asking.
- A repository with no record yet shows that it has never had one, rather than showing nothing.
- Nothing is ever blocked: no merge, check or pickup fails because a whole-repository review is overdue.
- The protocol suggests a whole-repository review when an epic or milestone closes; it stays advice.

## Why

An adopter merged about 30k lines across fifteen reviewed PRs before anyone read the repository as one system (#399). Per-PR review missed what only a whole read shows: one protocol implemented three times, a pure-logic crate drifting into a driver, safety policy coherent per PR and incoherent overall. A counter makes the gap visible cheaply; a forced review every N PRs would not be cheap (#399 *Cost note*).

## What Changes

1. `ateam review whole --at <sha> [--note <url>]` creates an annotated tag `review/whole/<YYYY-MM-DD>-<short sha>` at the reviewed commit and pushes it to `origin` (`--no-push` skips the push).
2. `ateam state sync`/`pickup` refresh counts first-parent commits and `git diff --shortstat` lines from the newest such tag to the default branch's remote tip, cached in `.ateam/state.json`.
3. The pickup line gains `since-review=<n> commits, +<a>/-<d> lines (<date>)` (under squash merge, one commit is one PR), or `since-review=never`; `ateam state status` JSON carries the same fields.
4. Protocol § 2 step 5 (Close) names an epic or milestone close as the point a whole-repository review is due, in the repository copy and both templates.

## Scope & Non-Goals

- **In-bounds:** `internal/state` (git snapshot, summary line, tag creation), `cmd/ateam/review.go` (subcommand wiring only), `openspec/specs/harness/spec.md` (one ADDED Requirement), `.agents/protocol.md` and `internal/bootstrap/templates.go`, README and `docs/daily-use.md`.
- **Non-goals:** running or structuring the review itself, a merge gate or nudge threshold, forge-issue records, counting lines per language, `internal/review/` (gate file) changes.

## Definition of Done

- Tests on a temporary git repository: no tag shows `never`; a tag then two first-parent commits shows `2 commits` and the literal `+/-` totals `git diff --shortstat` reports; the newest of two tags wins; `--no-push` leaves `origin` untouched. `task mutate:check` detects removal of each rule.
- The session-start hook still reads only the cache (no new git call on that path).
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
