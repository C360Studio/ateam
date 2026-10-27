## MODIFIED Requirements

### Requirement: Accurate MCP integration documentation

The MCP guide SHALL document the implemented tools and actions, prompts, resources, launch requirements, repository working directory, state/task/posting side effects, and verification steps. It SHALL distinguish local configuration checks, direct protocol tests, and real-client connection evidence.

#### Scenario: Following the MCP guide

- **WHEN** a user follows the setup and smoke-check instructions
- **THEN** examples match the exposed schema, do not advertise an unsupported sync action, and explain local-only handoff using post=false

#### Scenario: Posting side effect at the tool boundary

- **WHEN** an MCP client lists tools
- **THEN** the `ateam_state` `post` property states that an omitted or `true` value attempts a PR comment and `false` saves the handoff locally only, and the canonical handoff skill tells agents to pass `post` explicitly
