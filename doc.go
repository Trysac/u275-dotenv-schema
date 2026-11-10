// Package dotenvschema parses .env files strictly and validates them against a JSON schema.
//
// The dotenv grammar recognized by Parse is:
//
//   - Blank lines and lines whose first non-space character is '#' are skipped.
//   - An optional "export " prefix is allowed before the key.
//   - A key must match ^[A-Za-z_][A-Za-z0-9_]*$ and is followed by '='.
//   - Unquoted values are trimmed and terminate at the first " #" or "\t#" sequence.
//   - Single-quoted values are literal; no escape sequences are interpreted.
//   - Double-quoted values support \n, \t, \", and \\; the value may not span lines.
//   - After a closing quote, only whitespace or a comment may follow.
//   - Values may not exceed 65,536 bytes.
//   - A leading UTF-8 BOM is skipped and CRLF line endings are accepted.
//
// No variable expansion is performed.
package dotenvschema
