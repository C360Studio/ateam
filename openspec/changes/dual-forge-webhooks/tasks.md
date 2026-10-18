# Tasks: Dual-Forge Webhook & Branch Protection Sync

## 1. Foundation & Signature Verification

- [x] 1.1 Add constant-time HMAC-SHA256 signature verification helper (`gitea-webhook` benchmark fixture)
- [x] 1.2 Support GitHub (`gh`) and Gitea (`/api/v1` + `tea`) issue and PR board synchronization
- [x] 1.3 Seed canonical workflow labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`) via `ateam init`

## 2. Live Webhook Ingestion & Merge-Gate Rollout

- [ ] 2.1 Wire `POST /api/webhooks/gitea` and `POST /api/webhooks/github` into `internal/web/server.go`
  - Hold: awaiting owner decision on Gitea branch protection ruleset (#5) before enabling automatic status checks.
- [ ] 2.2 HALT: pause automatic PR merge-check rollout until Gitea webhook secret rotation lands in staging
- [ ] 2.3 Rate-limit Policy still open for unauthenticated webhook ping bursts on public Gitea instances
  - Hold: confirm whether per-IP fixed-window limiter in `internal/web` should default to 60 req/min.
- [~] 2.4 Skip GitHub Enterprise Server SAML webhook mapping for v0.1 (deliberate deferral to v0.2)
