## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and accepts ADR 0003 by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#179).
  Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [x] 2.1 Write `docs/adr/0003-roles-and-routes.md` (#179). Evidence: catalog rows cite `.agents/contracts/{architect,developer,reviewer}.md`; routes cite ADR 0002 tiers and bands (`.agents/review-policy.json`); `internal/review/policy.go:83-84` checked for the two route roles; ADR 0002 § 3 checked for comma-separated `implemented-by:` pairs.
- [x] 2.2 Pointers (#179). Evidence: `docs/model-review.md` status block, `.agents/README.md` § Configured model review, `.agents/protocol.md` § 2 Clarify Deliverable.
- [x] 2.3 Architect contract default (#179). Evidence: `.agents/contracts/architect.md` ¶ 3 and `architectContract` in `internal/bootstrap/templates.go`, byte-identical.
- [ ] 2.4 On approval: ADR 0003 Status set to `Accepted (owner, <date>)`, then archive (#179).
