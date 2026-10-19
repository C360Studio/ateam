# Change: Dual-Forge Webhook & Branch Protection Sync (`dual-forge-webhooks`)

implemented-by: gemini

## Why
Teams running `ateam` across both GitHub and self-hosted Gitea need real-time board card updates when draft PRs open, CI checks complete, or `status:needs-decision` / `status:blocked` labels change, without polling `gh` or `tea` on every UI refresh.

## What Changes
- Add HMAC-SHA256 webhook verification for GitHub (`X-Hub-Signature-256`) and Gitea (`X-Gitea-Signature`).
- Reconcile incoming issue/PR events directly into `.ateam/state.json`.
- Enforce branch-protection merge gates when OpenSpec `Hold:` items remain open.

## Impact
- Affected specs: `openspec/specs/harness/spec.md`
- Affected packages: `internal/state`, `internal/web`
