## ADDED Requirements

### Requirement: Gate reads retry transient forge failures

`scripts/gate-files.sh` SHALL try each forge read up to four times, waiting 5, 15, then 40 seconds between attempts, before exiting 2 naming the read. `ateam review gate`, for the `Merge check` and its probe, SHALL retry each read on the same schedule unless the forge answered with a 4xx status, and SHALL NOT retry an answer of the wrong shape. The probe step SHALL print the gate's output before exiting with its non-zero status.

#### Scenario: A read that fails briefly passes

- **WHEN** the pull request files read answers HTTP 500 twice and then lists the files
- **THEN** `Gate files` and `Merge check` reach the same verdict as with no failure

#### Scenario: A read that keeps failing stays unavailable

- **WHEN** the pull request files read answers HTTP 500 on every attempt
- **THEN** the check makes four attempts and exits 2 naming the read, never counting it as no gate file or a pass

#### Scenario: A 4xx answer is read once

- **WHEN** `ateam review gate` reads a proposal.md that answers HTTP 404, or Gitea branch protection that answers HTTP 403
- **THEN** it makes one attempt and treats the answer as it does today

#### Scenario: A failed probe names its read

- **WHEN** `ateam review gate --awaiting` exits non-zero in the probe step
- **THEN** the step log shows the gate's `merge-check: unavailable: …` line before the step fails with the same status
