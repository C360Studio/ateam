## ADDED Requirements

### Requirement: Owned Mercury bridge startup

The Mercury runner SHALL let the pinned bridge bind its own kernel-assigned loopback port. It SHALL obtain the actual port from bounded child startup output before making HTTP readiness requests, and SHALL retain the authenticated model-readiness check. It SHALL NOT reserve, release and subsequently reuse a guessed free port. Child exit, malformed or missing startup output, and cancellation SHALL not produce successful readiness or unbounded waiting. Existing provider usage and process cleanup boundaries SHALL remain intact.

#### Scenario: Concurrent listeners
- **WHEN** other local servers hold ports while the bridge starts
- **THEN** the bridge binds its own port zero and the runner probes only the actual loopback address announced after binding
- **AND** no speculative readiness probe is sent to an unrelated reserved-port server

#### Scenario: Startup evidence missing or invalid
- **WHEN** no valid complete pinned loopback startup line is retained within the diagnostic limit, or the child exits or is cancelled before readiness
- **THEN** startup fails within the attempt deadline and the child is cleaned up without starting native inference
