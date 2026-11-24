## ADDED Requirements

### Requirement: Whole-repository review marker

`ateam review whole --at <sha>` SHALL record a whole-repository review as an annotated `review/whole/<date>-<short sha>` tag at a default-branch commit and push it to `origin` unless `--no-push` is given. `ateam state pickup` SHALL print `since-review=` with the first-parent commits and diff lines from the newest such tag to the default branch's remote tip, or `never` without one, and SHALL never fail on it.

#### Scenario: No review recorded

- **WHEN** a repository has no `review/whole/` tag
- **THEN** the pickup line shows `since-review=never` and pickup exits 0

#### Scenario: Commits and lines since the newest review

- **WHEN** a `review/whole/` tag points at a default-branch commit and two first-parent commits adding 3 lines and deleting 1 follow it on the remote tip
- **THEN** the pickup line shows `since-review=2 commits, +3/-1 lines` with the tag's date
- **AND** with two tags, the counts start from the one latest in first-parent history

#### Scenario: Recording a review

- **WHEN** `ateam review whole --at <sha> --no-push` runs on a default-branch commit
- **THEN** it creates the annotated tag at `<sha>` and leaves `origin` unchanged
- **AND** a commit not on the default branch is refused with a non-zero exit and no tag
