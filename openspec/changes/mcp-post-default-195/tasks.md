## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#195).
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 The `post` property of the `ateam_state` schema describes the omitted/`true` PR-comment side effect and `false` as local-only, within the 300-token budget (#195).
- [ ] 2.2 The canonical handoff skill and its `templates.go` copy tell agents to pass `post:false` for local-only and `post:true` when posting is authorized; `ateam doctor` parity stays green (#195).
- [ ] 2.3 A test checks the `tools/list` `post` description, proven able to fail by `task mutate:check` (#195).
