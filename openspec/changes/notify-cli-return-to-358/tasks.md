## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit (#358).
  - Evidence: the owner approved in Enjoy on 2026-11-11 and asked Claude to write the header.

## 2. Notify on cli

- [ ] 2.1 On host `cli` with a non-empty `return_to`, exit 1, print the send line on stdout, and name `ATEAM_HOST` and the manifests in `.agents/hosts/` on stderr.
  - Premise: `cmd/ateam/reviewlaunch.go:448` returns 0 on `cli` before any output. The default branch of the notify switch already formats `notify: host %s has no notify …; send %q to %q`.
  - Acceptance: `TestReviewNotify` (`// spec: openspec/specs/model-review/spec.md#Requirement: Review notify text`) checks `cli` with and without `--return-to`. `task mutate:check` gives `detection` for restoring the unconditional `return 0`.
- [ ] 2.2 Add `--return-to <id>` to the abbreviated re-review line in `.agents/skills/review-pr/SKILL.md:35` and `internal/bootstrap/templates.go:526`.
  - Acceptance: `go test ./internal/bootstrap` and `ateam doctor --check` are green.

## 3. Close

- [ ] 3.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok`.
