## 1. Implementation

- [ ] 1.1 GitHub sync lists open issues and PRs uncapped and closed ones capped at 50, merged by number (#149).
- [ ] 1.2 Gitea sync pages `state=open` until a short page and reads `state=closed&limit=50`, merged by number (#149).
- [ ] 1.3 The pickup `synced` line names the closed-item cap (#149).
- [ ] 1.4 `forge_test.go`: 60 open PRs numbered 1 to 60 yield 60 cards, and a worktree on PR 1's branch labels `open PR #1`, on both forges. `task mutate:check` with the cap restored reports `detection` (#149).
