## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design OQ1 and OQ2 (#336). Hold: OPEN-Q proposal awaits owner approval

## 2. Policy key

- [ ] 2.1 Rename and alias in `internal/review/policy.go` (D1 A).
  - Premise: the field is at `policy.go:17`, validation at `:150-151` and `:188`, the strict key set comes from tags (`json.go:70-93`), and `ParsePolicy` is at `:104-110`.
  - Acceptance: `policy_test.go` table cases (`// spec: openspec/specs/harness/spec.md#Requirement: Reviewer checks policy key`) cover:
    - `reviewer_checks` only parses, and `required_checks` only parses to the same `ReviewerChecks`;
    - both keys (including `[]` for either) are rejected;
    - `reviewer_checks` without models is rejected with the new wording;
    - duplicate or padded names are rejected under either key.
  - `task mutate:check` reports `detection` for dropping the fold, dropping the both-keys check, and swapping `!= nil` for `len() > 0`.
- [ ] 2.2 Rename the reader at `internal/review/record.go:227-230` and its error text (D3).
  - Acceptance: the existing record tests pass with the new message. A record test asserts `reviewer check "task verify" lacks …`.
- [ ] 2.3 Move fixtures to `reviewer_checks` at `cmd/ateam/review_test.go:21`, `internal/bootstrap/review_test.go:11`, and `internal/web/routing_test.go:13`. Leave `internal/state/settings_test.go:57` as it is (it tests an unknown key in a different file).
  - Acceptance: `task test:unit` is green.

## 3. Doctor warning (if OQ1 = D2 A)

- [ ] 3.1 Add `review.PolicyUsesDeprecatedKeys(data)` and `bootstrap.CheckDeprecatedPolicyKeys(root)`, appended at `cmd/ateam/main.go:602`.
  - Acceptance: a doctor test warns on `required_checks`, and stays silent on `reviewer_checks`, on a tier-only policy, and when there is no policy. A mutation that forces the check off is detected.

## 4. Docs

- [ ] 4.1 `docs/model-review.md:38` example and one contrasting sentence linking `docs/adoption.md:95` (D3).
  - Acceptance: a reviewer reads the edit, and `task spec:check` passes.

## 5. Close

- [ ] 5.1 `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green on a clean tree. A not-weaker review record is posted at the final head. The owner approves the gate change at that head.
- [ ] 5.2 Archive this change before marking the PR ready. If OQ2 names a release, file the alias-removal issue in that milestone.
