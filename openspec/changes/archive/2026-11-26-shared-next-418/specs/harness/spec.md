## ADDED Requirements

### Requirement: Shared computed next issue

`ateam state next` SHALL name the first issue in the earliest open milestone's `Order:` line that is open, has no open `Depends on:` issue, carries neither `status:needs-decision` nor `status:blocked`, and is not claimed by an open PR, printing why each earlier issue was skipped. It SHALL name nothing, with a reason, when that milestone or the dependencies cannot be determined, and never an issue no `Order:` line places. Pickup SHALL show `next=` from the cache.

#### Scenario: Claimed and dependent issues are skipped

- **WHEN** the earliest milestone's description reads `Order: #1 -> #2 -> #3`, an open PR body says `Closes #1`, and #2 has a comment `Depends on: #1`
- **THEN** `ateam state next` names #3, with #1 skipped as claimed and #2 skipped as waiting on #1

#### Scenario: Milestone order follows After lines

- **WHEN** milestone B's description reads `After: A` and A has an open issue in its `Order:` line
- **THEN** the next issue comes from A, and none comes from B while A has an open ordered issue

#### Scenario: The order cannot be determined

- **WHEN** two open milestones with `Order:` lines both lack an `After:` line naming an open milestone
- **THEN** the pickup line shows `next=none (milestone order ambiguous)` and pickup exits 0

#### Scenario: Unplaced issues are listed, never named

- **WHEN** an open issue has no milestone, or is in a milestone but absent from its `Order:` line
- **THEN** `ateam state next` lists it as unplaced and never names it next
