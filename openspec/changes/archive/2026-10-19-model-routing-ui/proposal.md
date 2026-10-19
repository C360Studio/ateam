# Change: Restore the preserved dashboard and integrate model routing

Issue: https://github.com/C360Studio/ateam/issues/21
Draft PR: https://github.com/C360Studio/ateam/pull/22
Owner: Codex (`codex/model-routing-ui`)
implemented-by: codex

## Why

The CLI exposes configured model routing, but the dashboard cannot show developer/reviewer models, reasoning effort, or why a route is blocked.

## What Changes

Add a read-only authenticated `/api/model-routing` endpoint and the same `model_routing` summary in `/api/state`. Display a routing panel with exact model/provider/effort, eligibility and blocking reasons. Distinguish unconfigured, invalid and unavailable policy. Reuse `review.ParsePolicy` and `Policy.ValidateRouting`.

The user confirmed the unpushed `codex/ui-preserved-work` branch as the intended UI baseline. Restore its repository switcher, settings, Current Ops, canonical OpenSpec details and card drawers, and integrate routing with the selected repository. Repair repository persistence, stale-client writes, target selection and modal accessibility discovered during independent review.

## Scope and Risks

Touch the web server, focused routing reader/tests, dashboard types, a Svelte panel and rendered tests, plus model-review docs/specs. Existing API authorization and dashboard loading are callers. No dependency, policy editor, agent launch, runtime-model assertion or PR approval integration. Read failures must not hide the dashboard or imply eligibility. Bound policy reads and confine them to the selected repository. The restored settings add local repository registration and forge configuration; validate input and preserve explicit choices. Shared server selection must reject stale-client writes.

## Acceptance and Stop Boundary

Verify all five routing states, read authorization/method refusal, fresh reads after policy replacement/deletion, bounded/confined file reads, exact display/escaping and arbitrary effort strings. Run relevant gates, mutation checks, independent review and full preflight. Verify selected-repository histories, durable settings, configured forge targets, async response ordering and native dialogs. Deliver the preserved history and verified integration in the existing draft PR. The user explicitly authorized issue, branch and draft PR publication after the initial automatic approval rejection. Stop before merge.
