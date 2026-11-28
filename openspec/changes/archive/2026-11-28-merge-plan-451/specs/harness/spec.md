## ADDED Requirements

### Requirement: Coordinator merge plan

The shipped `coordinate` skill SHALL merge coordinated pull requests in the order they hand off `ready`, a pull request touching a gate file last, and SHALL release one approved pull request at a time. It SHALL request review of a pull request sharing a changed file with an earlier one only after that one has merged. The shipped developer contract SHALL arm auto-merge on a coordinated pull request only on the coordinator's message.

#### Scenario: Units sharing a file are reviewed in turn

- **WHEN** two coordinated pull requests change the same file and the later one hands off `ready` while the earlier one is open
- **THEN** the shipped `coordinate` skill tells the later one to wait, and requests its review only after the earlier one has merged and the later one has merged `main` into its branch

#### Scenario: Auto-merge waits for the coordinator

- **WHEN** a coordinated pull request's review record approves
- **THEN** the shipped developer contract has the developer archive the change and arm auto-merge only on the coordinator's message, and the shipped `coordinate` skill sends that message to one approved pull request at a time, telling the next to merge `main` and arm after each merge

#### Scenario: A gate-file pull request goes last

- **WHEN** an approved coordinated pull request touches a gate file
- **THEN** the shipped `coordinate` skill releases it only when no other approved pull request waits, and releases no other while it waits for the owner's approval

#### Scenario: A mid-run scope change reaches the owner

- **WHEN** `ateam state next` or the milestone's `Order:` line differs from the coordinator's last handoff
- **THEN** the shipped `coordinate` skill names the change in its next owner message
