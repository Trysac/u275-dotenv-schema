# Use a strict, line-oriented dotenv parser with schema-driven validation

Status: accepted (generated, pending human review)

## Context

The library must validate .env files in CI with accurate line numbers, keep the example file in sync, and never expose secret values. Existing dotenv parsers are too permissive and validation is often ad hoc.

## Decision

Implement a custom dotenv parser that strictly follows the documented grammar (no variable expansion, single and double quotes with escapes) and tracks line numbers. Validation uses a JSON schema with typed rules. Error messages contain only keys, never values.

## Consequences

The parser is self-contained and secure for untrusted input. Line numbers enable precise diagnostics. The schema loader validates its own defaults and compiles regexes once. The generated .env.example is derived from the same schema, eliminating drift.
