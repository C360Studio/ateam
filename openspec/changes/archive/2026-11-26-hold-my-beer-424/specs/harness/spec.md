## ADDED Requirements

### Requirement: Judge approvals under the hold-my-beer opt-in

A `Judged-by:` line in `proposal.md` SHALL count as approval only while `.agents/review-policy.json` sets `hold_my_beer: true`. It SHALL bind to the same digest as `Approved-by:`, which leaves both lines out, and SHALL never be shown as the owner's. While the opt-in is off, `spec-check` SHALL fail a filled `Judged-by:`.

#### Scenario: Judged approval with the opt-in on

- **WHEN** the policy sets `hold_my_beer: true`, `Approved-by:` is empty, and `Judged-by: codex (gpt-6-astra, high) sha256:<current digest>` is filled
- **THEN** `spec-check` passes, the queue labels the change judged, and `ateam/proposal` reads `judged by codex (gpt-6-astra, high)`

#### Scenario: Judged approval with the opt-in off

- **WHEN** the policy lacks `hold_my_beer` and `Judged-by:` is filled
- **THEN** `spec-check` exits 1 naming the change and `judged approval without hold_my_beer`

#### Scenario: The digest ignores both approval lines

- **WHEN** a `Judged-by:` or `Approved-by:` line is filled or emptied and nothing else changes
- **THEN** the digest `spec-queue` prints is unchanged

#### Scenario: Merge check names a judged change

- **WHEN** a code pull request touches a change whose `proposal.md` has an empty `Approved-by:` and a filled `Judged-by:`
- **THEN** the merge check notes that the change was judged by the named agent, not owner-reviewed, and its exit status is unchanged
