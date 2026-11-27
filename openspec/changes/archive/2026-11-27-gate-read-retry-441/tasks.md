# Tasks: gate-read-retry-441

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-27) sha256:022b5f5ebb8d`), written by claude on the owner's instruction in the #441 developer conversation; rulings D1 B, D2 A, D3 A, D4 A.

## 2. Implementation

- [x] 2.1 Shell retry (premise `scripts/gate-files.sh:34-56`, four reads; callers `:58,:62,:83,:97`): one helper runs a read, capturing each attempt's stdout separately so the partial pages of an unsuccessful `--paginate` attempt never reach the next, and waits per `ATEAM_GATE_RETRY_WAITS` (default `5 15 40`) between attempts. Accept: `gatefiles_test.go` cases on GitHub and Gitea fakes — a files read that errors twice then answers passes with the same verdict, one that errors on every attempt exits 2 naming the read with exactly 4 calls; mutation (cp + SHA-256) removing the retry loop is detected.
- [x] 2.2 Go retry (premise `internal/review/gate_github.go:24-33`, `gate_gitea.go:32-38`, `gate.go:144-167`): a package schedule `5s, 15s, 40s` honouring `ctx`; `gh` retries unless stderr has `(HTTP 4dd)`; Gitea retries a `*url.Error` or `forgeStatusError` ≥ 500 only. Accept: `gate_test.go` fake `gh` and `httptest` Gitea cases — 500 twice then answer passes; 500 every time exits 2 after 4 calls; a 404 on `fileAt` and a 403 on Gitea branch protection are read once; a decode error is not retried (`TestGiteaTransient`, with the 4xx and no-response literals). `task mutate:check` detects dropping the retry and dropping the 4xx exclusion.
- [x] 2.3 Probe output (premise `.github/workflows/merge-check.yml:61-64`, `.gitea/workflows/merge-check.yml:58-61`): `out=$(…) || { rc=$?; printf '%s\n' "$out"; exit "$rc"; }` in both. Accept: `TestGiteaWorkflowsTwinGitHub` and `mergecheck_test.go:905` pass; `TestProbeStepPrintsGateOutput` runs both probe steps under `bash -e` and asserts the gate's line is printed and its exit kept; reverting the GitHub step to the bare form is detected (cp + SHA-256).
- [x] 2.4 `task verify`, `ateam doctor --check`, `ateam bench --selftest` green at f9613f5 plus the runGitHub split; this change archived before ready. The owner's `gate-change-approved` at the final head is the `Gate files` check's, not a task here.
