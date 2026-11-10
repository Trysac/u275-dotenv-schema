// Package dotenvschema is a small, dependency-free Go library: Validate .env files against a JSON schema in Go: strict dotenv parsing, typed rules for int, bool, enum, url, duration and pattern, secret-safe messages and a generated .env.example
//
// The public API is every exported identifier of this root package. Functions are pure: they do
// no I/O, keep no global mutable state and report invalid input through returned errors.
package dotenvschema
