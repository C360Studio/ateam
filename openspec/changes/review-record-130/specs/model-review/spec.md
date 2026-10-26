## ADDED Requirements

### Requirement: Review record mode

`.agents/review-policy.json` SHALL accept an optional `review_record` of `pr-comment` (the default when absent) or `aip` and reject other values. Under `pr-comment`, `ateam init` output SHALL be unchanged. Under `aip`, init, review setup and doctor SHALL name the archived `review-implementation.md` with `Verdict: PASS` and a matching `approval.md` as the review record, with no `reviewed-by:` PR comment and no merge-check.

#### Scenario: Default record mode is unchanged

- **WHEN** the policy has no `review_record`, or sets `pr-comment`, and `ateam init` runs
- **THEN** every generated file is byte-identical to the output with no policy, and `.agents/protocol.md` still asks for a PR comment starting `reviewed-by: <agent> (<model>, <effort>)`

#### Scenario: AIP record mode

- **WHEN** the policy sets `review_record: aip` and `ateam init` or `ateam review setup` runs
- **THEN** the protocol, both rule rows and the reviewer contract name the archived `review-implementation.md` with `Verdict: PASS` and a matching `approval.md`, say ateam agents never post a `reviewed-by:` PR comment and that `task merge:check` does not apply, no longer mention `Reviewed at`, every other generated file is unchanged, and `ateam doctor --check` passes

#### Scenario: Unknown record mode

- **WHEN** the policy sets `review_record` to any value other than `pr-comment` or `aip`
- **THEN** policy validation rejects it and `ateam init` fails without writing a file

### Requirement: Reviewer run from a review record

`ateam review compare --reviewer-record <path>` SHALL take the reviewer run from the Reviewer field (`<agent> (<model>, <effort>)`) of a REVIEW-002 implementation record and apply the tier-and-band comparison. A record without exactly one Verdict section, `Stage: implementation`, `Verdict: PASS` and parseable Reviewer, or without a sibling `approval.md` row `implementation` holding the record's SHA-256, SHALL be a finding (exit 1). merge-check SHALL NOT read file records.

#### Scenario: Reviewer run from an approved record

- **WHEN** `ateam review compare --implementer <run>... --reviewer-record <path>` names a record with `Verdict: PASS`, a Reviewer starting `<agent> (<model>, <effort>)`, and a sibling `approval.md` whose `implementation` row holds the record's SHA-256
- **THEN** compare applies the tier-and-band comparison to that reviewer run and exits 0 when it is not weaker, 1 when it is weaker

#### Scenario: Rejected record

- **WHEN** the record's verdict is not `PASS`, `approval.md` is missing or holds no `implementation` row with the record's digest, the Reviewer value names no `claude`, `codex` or `gemini` run, or the Verdict section or one of its fields is missing or repeated
- **THEN** compare prints each finding and exits 1; an unreadable record exits 2, and giving both `--reviewer` and `--reviewer-record` exits 2
