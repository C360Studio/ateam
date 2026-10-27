# Change Proposal: `roles-and-routes-179`

Approved-by:

## Why

The repository documents developer and reviewer contracts and ranks runs by tier and band (ADR 0002), but not the role catalog itself, the planning and judgment roles, or the default route per task class (#179, epic #180). That text lives only in the owner's private instructions, so an agent in this repository cannot read which tier drafts a design or when a `workhorse` developer is allowed.

## What Changes

- New `docs/adr/0003-roles-and-routes.md` (Status: Proposed until the owner accepts it with this proposal): catalog (architect, developer, reviewer, judge: contract, may write, never), default tier and band per role, the four conditions for a `workhorse` developer, escalation and below-default recording.
- Pointers: `docs/model-review.md` status block, `.agents/README.md` § Configured model review, the protocol's Clarify Deliverable line ("and which role drafts it").
- `.agents/contracts/architect.md` and its `ateam init` source in `internal/bootstrap/templates.go`: replace "pending ADR 0003" with the default the ADR sets.

## Non-goals

- An `architect` or `judge` route in the policy schema or `ateam review route` (`internal/review/`, a gate file; filed after acceptance).
- Mechanical enforcement of any route but the reviewer's, which the live gate already checks.
- The two items carried from #178 (`architect.md` in `review setup` drift; `rules: design:` in the init `config.yaml`): each is an `internal/bootstrap` behavior change for adopters, not an ADR pointer; filed as follow-ups.
- The adopter protocol template's Clarify line and any ADR text in `ateam init` output.

## Definition of Done

- The owner accepts the ADR by approving this proposal; the ADR's Status reads `Accepted (owner, <date>)` before archive.
- `task verify` green on a clean tree; `ateam doctor --check` passes (architect contract byte-equal to its bootstrap source).
- Every `file:line` and ADR cross-reference in the ADR opened and checked.
