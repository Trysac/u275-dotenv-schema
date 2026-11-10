# dotenv-schema

[![Go Reference](https://pkg.go.dev/badge/github.com/Trysac/u275-dotenv-schema.svg)](https://pkg.go.dev/github.com/Trysac/u275-dotenv-schema)
[![CI](https://github.com/Trysac/u275-dotenv-schema/workflows/ci/badge.svg)](https://github.com/Trysac/u275-dotenv-schema/actions)

Validate `.env` files against a JSON schema in Go. Strict dotenv parsing, typed rules for `int`, `bool`, `enum`, `url`, `duration` and pattern, secret-safe error messages, and a generated `.env.example`.

## Installation

Requires Go >= 1.23. The package is `dotenvschema` at `github.com/Trysac/u275-dotenv-schema`.

```sh
go get github.com/Trysac/u275-dotenv-schema
go install github.com/Trysac/u275-dotenv-schema/cmd/dotenv-schema@latest
```

## Usage

```go
package main
import ("fmt"; "strings"; dotenvschema "github.com/Trysac/u275-dotenv-schema")
func main() {
  s, _ := dotenvschema.LoadSchema(strings.NewReader(`{"vars":{"PORT":{"type":"int","min":1,"max":65535}}}`))
  e, _ := dotenvschema.Parse(strings.NewReader("PORT=70000\n"), dotenvschema.Options{})
  for _, i := range dotenvschema.Validate(e, s) { fmt.Printf("line %d: %s %s %s\n", i.Line, i.Code, i.Key, i.Message) }
}
```

CLI: `go run ./cmd/dotenv-schema --schema schema.json .env` and `go run ./cmd/dotenv-schema --schema schema.json --example > .env.example`. Exit code `0` when valid, `1` when issues are found, `2` on usage or I/O error.

## Public API

- `Parse(r io.Reader, opts Options) ([]Entry, error)` parses dotenv input. `Entry` is `{Key, Value string; Line int}`. `Options` has `MaxBytes` (default `1_048_576`) and `MaxLines` (default `10_000`). Values are at most `65_536` bytes. Blank lines and comment lines are skipped; an optional `export ` prefix is accepted; keys match `^[A-Za-z_][A-Za-z0-9_]*$`; `=` is required. Unquoted values are trimmed and cut at the first ` #` or tab-`#`; single-quoted values are literal; double-quoted values support `\n`, `\t`, `\"`, `\\` and may not span lines. A UTF-8 BOM is skipped and CRLF is accepted. There is no variable expansion.
- `LoadSchema(r io.Reader) (Schema, error)` reads at most 1 MiB and at most 1000 vars, rejects unknown JSON fields, compiles patterns once, and validates defaults. `Schema` is `{Vars map[string]Var; AllowUnknown bool}`; `Var` has `Type` (`string`, `int`, `bool`, `enum`, `url`, `duration`), `Required`, `Secret`, `Default *string`, `Min`, `Max *int64`, `Values []string`, `Pattern string`, `MinLength`, `MaxLength *int`, with camelCase JSON tags.
- `Validate(entries []Entry, s Schema) []Issue` returns `Issue{Code, Key string; Line int; Message string}` sorted by `Line`, then `Key`, then `Code`. Messages never contain values: `E001` required, `E002` type, `E003` range or length, `E004` pattern, `E005` enum, `E006` unknown key, `E007` duplicate key.
- `Example(s Schema) string` returns vars sorted by name, each preceded by `# <type>[, required][, secret]`, then `KEY=<default, or empty for secrets and vars without default>`.
- `Run(args []string, stdout, stderr io.Writer) int` implements the CLI: `--schema FILE` required, `--json` issues as a JSON array, `--example` prints `Example(schema)`, and file `-` reads stdin.

Errors: `ErrInputTooLarge`, `ErrSyntax`, and `ErrSchema` are sentinels. `*SyntaxError{Line int; Reason string}` unwraps to `ErrSyntax`.

## Why not typed config loaders or environment doctors

This utility validates the `.env` file itself in CI with line numbers and keeps `.env.example` in sync from the same schema. It does not read the real environment, does not expand variables, and does not load values into the process environment.

## Documentation

See the [package documentation](https://pkg.go.dev/github.com/Trysac/u275-dotenv-schema) and [docs/SECURITY.md](docs/SECURITY.md).
