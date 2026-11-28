# Tasks: board-host-check

## 1. Approval

- [ ] 1.1 Owner approves the proposal. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Host and Origin check in the tokenless branch of `authorizeAPIRequest` (premise: it authorises on `isLoopbackRequest(r.RemoteAddr)` alone, `internal/web/server.go:129-152`; callers are `validateReadRequest` and `validateMutationRequest`, including `internal/web/routing.go:72`). Accept: `TestTokenlessBoardAnswersOnlyLoopbackHosts`, `TestTokenlessBoardRefusesForeignOrigin`, `TestTokenedBoardAcceptsAnyHost`; existing tests that authorised by peer address alone now send a loopback Host.
- [ ] 2.2 Approve branch validation and the Gitea escape (premise: the handler only trims `branch`, `internal/web/server.go:731-735`; `giteaFiles.Head` puts it unescaped into the path, `internal/state/forgefiles.go:307`). Accept: `TestApproveRefusesInvalidBranchNames`, `TestValidBranchNameAcceptsGitBranches`, `TestGiteaHeadEscapesTheBranchPerSegment`.
- [ ] 2.3 Harness spec delta applied; one sentence each in `README.md` and `docs/adoption.md`; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
