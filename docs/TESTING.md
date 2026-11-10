# Testing

## Running

No test files are present in the supplied code. For a checkout, run `go test -cover ./...`.

## Policy

The specification defines acceptance cases that tests must cover: dotenv grammar and limits, schema loading and defaults, every issue code E001-E007, secret-safe messages that never contain values, exact `Example` output, and CLI `Run` exit codes. Tests must not print or assert secret values.

## Tiers

The supplied code contains no CI conformance setup and no `testdata/` fixtures. Do not add a conformance tier unless an external tool is introduced by an accepted change.

## Layout

When tests exist, keep `<file>_test.go` next to the tested file and fixtures in `testdata/`.
