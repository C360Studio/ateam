# Delta Spec: Dual-Forge Webhook & Hold Enforcement

## Purpose
Extend the `ateam` harness specification with authenticated GitHub and Gitea webhook ingestion and automatic `Hold:` merge blocking.

## Requirements

### Requirement: Constant-Time Forge Webhook Verification
The HTTP server SHALL verify `X-Hub-Signature-256` and `X-Gitea-Signature` headers using constant-time `hmac.Equal` before mutating `.ateam/state.json`.

#### Scenario: Valid Gitea webhook payload
- **WHEN** `POST /api/webhooks/gitea` receives a payload signed with `ATEAM_WEBHOOK_SECRET`
- **THEN** the matching `BoardCard` column and labels update in `.ateam/state.json`
