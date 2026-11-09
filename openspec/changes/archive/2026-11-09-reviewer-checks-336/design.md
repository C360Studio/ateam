# Design: `reviewer-checks-336`

## Context

Inventory taken at `e4851b4`; every line below was opened this session.

- `internal/review/policy.go:17` declares `RequiredChecks []string` with the tag `json:"required_checks,omitempty"`. `:150-151` rejects the key without `models` ("routes, owners, review_authors and required_checks require models"). `:188` runs `uniqueNames("required_checks", …)`.
- `internal/review/record.go:227-230` is the only reader. Each listed name needs `reviewer`-sourced `passed` evidence at the current head, otherwise the error is `required check %q lacks reviewer-executed passing evidence at current head`.
- `internal/review/json.go:15-31` decodes strictly. `checkObject` (`:70-93`) builds its key set from the struct's `json` tags and rejects any unknown or duplicate key. A tag therefore makes a key legal, and a removed tag makes existing policies fail to parse.
- `ParsePolicy` (`policy.go:104-110`) has 10 non-test callers (cmd/ateam ×4, bootstrap ×3, gate, state/forge, web/routing). All of them go through it, so a fold inside `ParsePolicy` reaches every caller.
- Nothing in `ateam` marshals a `Policy` back out (the `json.Marshal` calls in `internal/review` are only in tests and the gate's own payloads). The alias field cannot leak into written files.
- `ATEAM_REQUIRED_CHECKS` is unrelated and unchanged. It is read at `cmd/ateam/review.go:190-194` and set in both `merge-check.yml` workflows (`.github/…:102`, `.gitea/…:103`).
- Docs: `docs/model-review.md:38` shows the key. `docs/adr/0002-model-tiers-and-effort-bands.md:13,71` names it historically.
- Fixtures using the key: `cmd/ateam/review_test.go:21`, `internal/review/policy_test.go:11,73-74`, `internal/bootstrap/review_test.go:11`, `internal/web/routing_test.go:13`. `internal/state/settings_test.go:57` uses it only as an unknown key for `.agents/ateam.json`.
- This repo's own `.agents/review-policy.json` uses tiers only and sets no checks key, so it is unaffected.
- `ateam doctor` collects warnings at `cmd/ateam/main.go:601-622`. The policy is parsed for doctor at `internal/bootstrap/review.go:179`.

## Goals / Non-Goals

Goal: one unambiguous name, existing adopter policies keep working for a release, and adopters are told to move. Non-goals: see `proposal.md`.

## Decisions

### D1. How the alias parses

- **A (recommended): a second struct field.** Add `ReviewerChecks []string \`json:"reviewer_checks,omitempty"\`` plus an undocumented `DeprecatedRequiredChecks []string \`json:"required_checks,omitempty"\``. `ParsePolicy` rejects the policy when both are non-nil, then folds the deprecated field into `ReviewerChecks` and clears it before `ValidatePolicy`. This costs about 10 lines. `checkObject` needs no change, because the tag keeps the key legal. Presence can be tested with `!= nil` because `checkValue` rejects `null` (`json.go:41-43`), so `"required_checks": []` still counts as set. What it forecloses: the exported `Policy` gains one field until the alias is removed.
- **B: a custom `UnmarshalJSON` on `Policy`.** This keeps the struct clean, but `checkObject` validates keys from the tags before `json.Unmarshal` runs. Without a tag the old key would be rejected first, so the strict checker would need a special case in a gate file. That is more code with a wider blast radius.
- **Adopter seam:** if an adopter does nothing, their `required_checks` policy keeps parsing and enforcing exactly as before. If they set both keys, `ateam review setup` / doctor fail with `set reviewer_checks or required_checks, not both`.

### D2. Telling adopters to migrate

- **A (recommended): a doctor warning.** `bootstrap.CheckDeprecatedPolicyKeys(root)` is appended at `main.go:602`. It reports `.agents/review-policy.json: required_checks is deprecated; rename it to reviewer_checks` when the deprecated field was set. That needs `ParsePolicy` to report the fact. Either keep the field populated after the fold and have `ValidatePolicy` ignore it, or add a tiny `PolicyUsesDeprecatedKeys(data)` that decodes only that field. The second keeps `Policy` normalized and is chosen. Cost: about 15 lines plus a test.
- **B: a docs note only.** This costs nothing in code, but an adopter who never rereads the docs would hit a hard parse failure when the alias is removed.

### D3. Wording

`policy.go:151` and `:188` and `record.go:229` say `reviewer_checks` / `reviewer check %q`. The `docs/model-review.md:38` example uses `reviewer_checks`, and one sentence there contrasts it with `ATEAM_REQUIRED_CHECKS` and links `docs/adoption.md:95`. ADR 0002 stays as written because it is a historical record.

## Risks / Trade-offs

- **Stored digests.** `PolicyDigest` hashes raw bytes (`policy.go:113`). An adopter who renames the key changes the digest, and dormant-path evidence bound to the old digest goes stale. That is the intended behaviour for a policy edit, and nothing in the active merge-check path uses it.
- **Gate file.** Every edit is under `internal/review/` except bootstrap, cmd and docs, so the PR needs the owner's `gate-change-approved` at the final head.

## Migration Plan

1. Ship the alias and the doctor warning in v0.4.0.
2. Remove the alias in the release the owner picks (OQ2). A follow-up issue tracks that, and removal is a one-tag deletion plus a test flip.

## Open Questions

1. **OQ1 (owner ruling):** add the doctor warning (D2 A, recommended) or document the rename only (D2 B)?
2. **OQ2 (owner ruling):** which release removes `required_checks`? The recommendation is v0.5.0, filed as an issue in that milestone when this merges.
