# Change Proposal: `review-instructions-179`

Approved-by:

## Why

The instructions describe two review systems as if both were live (#179, v0.3.3 hardening acceptance). This repository's `.agents/review-policy.json` declares tiers and effort bands only, so `ateam review route` exits 1 by design. Yet protocol § 4 says to validate the route before implementation and that the reviewer must outrank every implementer, and the developer contract asks unconditionally to confirm the route. The evidence-refresh text says any new commit invalidates approval, while the live gate keeps a record across clean `main` merges and docs-only commits. `docs/model-review.md`'s status block already describes the live gate correctly; the instructions disagree with it.

## What Changes

- `.agents/protocol.md` § 4: state the live merge gate first (not weaker by tier then band, `cross_provider` only when set, record freshness per Gate 3), then the configured route / `ateam review check` path as conditional on a policy that declares models and routes, carrying its strict superiority, exact-head binding and ceiling exception.
- `.agents/contracts/developer.md` and `reviewer.md`, with their `ateam init` / `review setup` sources in `internal/bootstrap/review.go`: confirm a route only when one is configured; fresh review after a commit that is neither a clean `main` merge nor docs-only, with the stricter rule kept for the configured path.
- The adopter protocol § 4 template in `internal/bootstrap/templates.go`: lead with the live gate; mark route setup, `review route` and `review check` optional.
- `.agents/README.md` § Configured model review: one sentence saying a tier-only policy is valid and configures no routes.

## Non-goals

- The role catalog and routes (ADR 0003): the rest of #179, a later PR.
- Any gate file, gate behavior, model ranking or policy change; `#192`'s merge-check rerun sequencing.
- `architect.md` in `review setup` drift and `rules: design:` in the init `config.yaml` (carried from #178 to the ADR slice).

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor --check` parity green.
- Repo contracts byte-equal their bootstrap sources (`TestReviewSetup*` / doctor drift).
- Trace recorded in `tasks.md`: the tier-only workflow reaches merge with no route or exception step; an explicitly configured route still reads `review route` → `review check` with superiority, exact-head binding and the ceiling exception.
