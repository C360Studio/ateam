# Change Proposal: `state-help-197`

Approved-by:

## Why

`ateam --help` lists the `state` subcommands, but `ateam state --help` and `ateam state -h` exit 1 with `ateam state: unknown subcommand "--help"` (#197, seen in the Codex dogfood pass at `2d7eeef`). An agent that asks for help gets an error. `runState` in `cmd/ateam/main.go` has no help case, so its `switch` treats the flag as an unknown subcommand.

## What Changes

- `runState` treats `-h`, `--help`, and `help` as help. It prints the `state` subcommands, one line each, and exits 0. The usage text is a plain string in the style of `printUsage`.
- An unknown subcommand still exits 1. Its error line gains `run 'ateam state --help' for subcommands`.
- Help returns before any state read or write, so it does not refresh or write `.ateam/state.json`, take or clear the writer lease, sync the forge, or post comments.
- The "Durable shared state and five-field handoff" requirement in the harness spec gains one scenario for read-only help.

## Track

`quick`. The change modifies one existing requirement and adds none. It touches only `cmd/ateam` (no exported surface, boot path, durability, or gate file).

## Non-goals

- Per-subcommand help (`ateam state pickup --help` already prints its own `flag` usage).
- A CLI framework, a reorganization of commands, or shared usage tables.
- Help for other top-level commands.

## Definition of Done

- `task verify` is green on a clean tree, and `ateam doctor` parity is green.
- A test in `cmd/ateam/main_test.go` covers four cases. `state --help`, `-h`, and `help` each exit 0 and name every subcommand. They leave no `.ateam/state.json` in an empty temp repo. An unknown subcommand exits 1 and points to `--help`. `task mutate:check` shows that the test can fail.
