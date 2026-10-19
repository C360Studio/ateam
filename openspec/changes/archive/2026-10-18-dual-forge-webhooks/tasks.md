# Tasks: Dual-Forge Webhook & Branch Protection Sync

implemented-by: gemini

## 1. Foundation & Signature Verification

- [x] 1.1 Add constant-time HMAC-SHA256 signature verification helper (`gitea-webhook` benchmark fixture)
- [x] 1.2 Support GitHub (`gh`) and Gitea (`/api/v1` + `tea`) issue and PR board synchronization
- [x] 1.3 Seed canonical workflow labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`) via `ateam init`

## 2. Live Webhook Ingestion & Merge-Gate Rollout

- [x] 2.1 Wire `POST /api/webhooks/gitea` and `POST /api/webhooks/github` into `internal/web/server.go`
- [x] 2.2 Report `merge_allowed` and `active_holds` in webhook responses for PR merge-check enforcement
- [x] 2.3 Enforce 60 req/min per-IP fixed-window rate limiter on webhook endpoints in `internal/web/webhook.go`
- [~] 2.4 Skip GitHub Enterprise Server SAML webhook mapping for v0.1 (deliberate deferral to v0.2)
