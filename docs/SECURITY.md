# Security notes

## Threat model

### Defended

- Malformed dotenv input: `Parse` rejects invalid syntax with `ErrSyntax` / `*SyntaxError{Line, Reason}` and returns `ErrInputTooLarge` when input exceeds `MaxBytes` (default `1_048_576`) or `MaxLines` (default `10_000`). Individual values are limited to `65_536` bytes.
- Oversized schema: `LoadSchema` accepts at most 1 MiB and at most 1000 vars, rejects unknown JSON fields, and returns `ErrSchema` for invalid regex patterns or defaults that fail their own type.
- Secret leakage in diagnostics: `Validate` issues and CLI output contain codes, keys and line numbers, never values. `Example` emits an empty value for vars marked `secret` and for vars without a default.
- Regex handling: patterns are compiled with RE2 via the standard `regexp` package once in `LoadSchema`.
- Hostile values are not expanded: parsing is line-oriented and does not perform shell-like variable expansion.

### Not defended

- The library cannot prevent secrets from being committed to a `.env` file.
- It does not decrypt secrets, redact caller logs, or guarantee that a value is safe for a downstream service; it checks only the declared type, range, length, pattern, enum, URL or duration rule.
- It does not load values into the process environment or read the real environment.
- The CLI reads files or stdin supplied by the caller; protecting those paths and their permissions is outside this library.

## Publishing

Releases are git tags served through the Go module proxy and checksum database. No registry token is stored in the repository.

To report a vulnerability, see [SECURITY.md](../SECURITY.md).
