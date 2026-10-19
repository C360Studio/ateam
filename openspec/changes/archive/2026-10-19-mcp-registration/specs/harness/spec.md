## ADDED Requirements

### Requirement: Client-readable MCP registration

ATeam initialization SHALL place MCP registration in the supported configuration surface for each supported client, preserve unrelated existing settings, and document any required migration or manual setup. Local parity checks SHALL validate the expected registration structure without representing that result as proof of a live client connection.

#### Scenario: Codex MCP discovery

- **WHEN** ATeam is initialized for Codex in a trusted disposable project with its executable available
- **THEN** the registration uses Codex's supported MCP configuration surface and client discovery can identify the ATeam server without relying on an entry in a hooks file

#### Scenario: Existing client configuration

- **WHEN** initialization encounters existing client settings and servers
- **THEN** unrelated configuration is preserved and any required manual migration is explained instead of silently overwriting settings

### Requirement: Accurate MCP integration documentation

The MCP guide SHALL document the implemented tools and actions, prompts, resources, launch requirements, repository working directory, state/task/posting side effects, and verification steps. It SHALL distinguish local configuration checks, direct protocol tests, and real-client connection evidence.

#### Scenario: Following the MCP guide

- **WHEN** a user follows the setup and smoke-check instructions
- **THEN** examples match the exposed schema, do not advertise an unsupported sync action, and explain local-only handoff using post=false
