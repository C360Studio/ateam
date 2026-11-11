## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit (#358).
  - Evidence: the owner approved in Enjoy on 2026-11-11 and asked Claude to write the header.

## 2. Notify on cli

- [x] 2.1 On host `cli` with a non-empty `return_to`, exit 1, print the send line on stdout, and name `ATEAM_HOST` and the manifests in `.agents/hosts/` on stderr.
  - Premise: `cmd/ateam/reviewlaunch.go:448` returns 0 on `cli` before any output. The default branch of the notify switch already formats `notify: host %s has no notify …; send %q to %q`.
  - Acceptance: `TestReviewNotify` (`// spec: openspec/specs/model-review/spec.md#Requirement: Review notify text`) checks `cli` with and without `--return-to`. `task mutate:check` gives `detection` for restoring the unconditional `return 0`.
  - Evidence: `reviewNotify` (`cmd/ateam/reviewlaunch.go`) on `cli` returns 0 only with no target. Otherwise it prints the send line, names `ATEAM_HOST` and `hostManifestList` on stderr, and exits 1. A `TestReviewNotify` case checks exit 1, no calls, the exact stdout and the stderr hint `(one of: cmdhost, mcphost)`. `task mutate:check` gives `detection` for restoring the unconditional `return 0`, and the source SHA-256 was unchanged.
- [x] 2.2 Add `--return-to <id>` to the abbreviated re-review line in `.agents/skills/review-pr/SKILL.md:35` and `internal/bootstrap/templates.go:526`.
  - Acceptance: `go test ./internal/bootstrap` and `ateam doctor --check` are green.
  - Evidence: both copies now read `ateam review request --reviewer-session <your id> --return-to <id> <n>`. `go test ./internal/bootstrap` passes, and `ateam doctor --check` reports `violations: null`. `docs/adoption.md` notes the `cli` exit.

## 3. Close

- [x] 3.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok`.
  - Evidence: archived with `node_modules/.bin/openspec` 1.14.3. `task verify` exit 0 on a clean tree, with `openspec: strict ok` naming `node_modules/.bin/openspec`.
