## MODIFIED Requirements

### Requirement: Inline shortcut inventory

Shortcut scanning SHALL report the source path, line, limit, and revisit trigger from supported source comment markers.

#### Scenario: Source marker

- **WHEN** a source file contains a shortcut comment with a comma-separated revisit trigger
- **THEN** the inventory preserves its location and both fields

#### Scenario: Rust and Python sources outside build output

- **WHEN** `.rs` or `.py` files carry shortcut comments, including copies under `target`, `.venv`, or `__pycache__` directories at any depth
- **THEN** the inventory reports the source files and omits every file under those directories

#### Scenario: Rust doc comments and bracketed commas

- **WHEN** a shortcut follows `///`, `//!`, or `/// //`, or its limit contains a comma inside parentheses or brackets
- **THEN** the inventory reports it and splits the limit from the revisit trigger at the first comma outside those brackets
