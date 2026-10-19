## MODIFIED Requirements

### Requirement: Native agent evidence
The harness SHALL provide a Gemini CLI adapter with explicit API-key authentication, isolated attempts, bounded redacted output, CLI identity and inspectable execution settings. It SHALL normalize complete CLI-reported input, candidate and thought counts without double-counting cache. Incomplete accounting SHALL remain unknown, failures SHALL retain evidence, and unavailable backend-resolved settings SHALL remain unknown.

#### Scenario: Provider reports cached tokens
- **WHEN** Gemini reports prompt, cached prompt, candidate and thought counts
- **THEN** normalized input includes cached input once and normalized output includes candidate plus thought tokens once

#### Scenario: Failed or interrupted execution
- **WHEN** a native attempt cannot return a complete accounting record
- **THEN** its failure and retained output remain visible and no further native attempt starts with unknown usage

#### Scenario: Failed API requests lack complete accounting
- **WHEN** API request/error counters are absent or any request failed
- **THEN** normalized usage remains unknown even if a later request succeeds, and no further attempt starts

## ADDED Requirements

### Requirement: Fixed-model harness treatments
The native pilot SHALL use gemini-3.8-flash with the same requested model and generation settings for both arms. A/B SHALL vary explicit harness instruction treatment; A/A SHALL hold that treatment identical. Observed usage from an unexpected model or missing per-model identity SHALL stop the pilot and SHALL NOT contribute a valid same-model quality result. Model routing and role-to-model defaults SHALL remain outside this benchmark change.

#### Scenario: Provider silently changes a model
- **WHEN** native per-model statistics include a model other than the configured model
- **THEN** the attempt retains its accounting evidence but is marked an execution error and further calls stop
