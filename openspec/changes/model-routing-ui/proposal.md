# Change: Read-only model routing dashboard

Owner: Codex (`codex/model-routing-ui`)
implemented-by: codex

## Why

The CLI exposes configured model routing, but the dashboard cannot show developer/reviewer models, reasoning effort, or why a route is blocked.

## What Changes

Add a read-only authenticated `/api/model-routing` endpoint and the same `model_routing` summary in `/api/state`. Display a routing panel with exact model/provider/effort, eligibility and blocking reasons. Distinguish unconfigured, invalid and unavailable policy. Reuse `review.ParsePolicy` and `Policy.ValidateRouting`.

## Scope and Risks

Touch the web server, focused routing reader/tests, dashboard types, a Svelte panel and rendered tests, plus model-review docs/specs. Existing API authorization and dashboard loading are callers. No dependency, policy editor, agent launch, runtime-model assertion or PR approval integration. Read failures must not hide the dashboard or imply eligibility. Bound policy reads and confine them to the served repository.

## Acceptance and Stop Boundary

Verify all five routing states, read authorization/method refusal, fresh reads after policy replacement/deletion, bounded/confined file reads, exact display/escaping and arbitrary effort strings. Run relevant gates, mutation checks, independent review and full preflight. Deliver locally verified work; issue/PR publication requires authorization after automatic approval review rejected issue creation.
