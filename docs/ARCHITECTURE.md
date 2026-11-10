# Architecture

The library is split into three main components:

- **Parser** (`dotenv.go`): strict, line-oriented parser with limits on total size, line count, and individual value length. It returns `Entry` values with line numbers for accurate diagnostics.
- **Schema loader** (`schema.go`): reads and validates a JSON schema, compiling regex patterns and checking defaults. The schema is stored with compiled regexes in an unexported field to avoid recompilation.
- **Validator** (`schema.go`): compares entries against the schema, producing a sorted `[]Issue`. Issues never include values, and messages use a fixed format.

The CLI (`cli.go`, `cmd/dotenv-schema`) wraps these components, reading from files or stdin and formatting output either line-by-line or as JSON.

## Design decisions

- **No variable expansion**: the dotenv grammar is deliberately strict and does not perform shell-like expansion. This keeps parsing predictable and avoids security issues with untrusted input.
- **Secret safety**: validation messages include only keys, never values, and the `Secret` flag further suppresses defaults in generated examples.
- **Limits**: every input has an explicit size limit; parsers return a typed error when exceeded.
