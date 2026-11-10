# Project record

## Objective

Validate .env files against a JSON schema in Go: strict dotenv parsing, typed rules for int, bool, enum, url, duration and pattern, secret-safe messages and a generated .env.example.

## Scope

- In scope: strict dotenv parsing with line-numbered `Entry` values; JSON schema loading with unknown-field rejection and default validation; typed validation for `string`, `int`, `bool`, `enum`, `url` and `duration`; pattern, range and length checks; issue collection with codes E001-E007; secret-safe messages; `.env.example` generation from the schema; CLI `Run` with file, stdin, JSON and example modes.
- Out of scope: variable expansion, multi-line values, loading values into the process environment, and decrypting secrets.

## Timeline

| Date | Event |
|---|---|
| 2026 | Repository created from the factory utility template. |

## Effort metrics

Measured values only; a value that has not been measured stays "pending measurement".

| Metric | Value |
|---|---|
| Tokens (input / output) | 631609 / 59203 |
| Hours (human) | pending measurement |
| Cost (USD) | 0.0804 |

## Factory run u275 (measured from API usage)

| Date (UTC) | Phase | Model | Batch | Input | Output | Cache write | Cache read | Cost (USD) |
|---|---|---|---|---|---|---|---|---|
| 2026-11-10 | generation | deepseek-v4-pro | yes | 1840 | 22486 | 0 | 3328 | 0.0458 |
| 2026-11-10 | fix-1 | deepseek-flash | no | 19873 | 6208 | 0 | 197760 | 0.0073 |
| 2026-11-10 | fix-2 | deepseek-flash | no | 10251 | 2835 | 0 | 59136 | 0.0034 |
| 2026-11-10 | audit | deepseek-flash | no | 26458 | 16310 | 0 | 210560 | 0.0144 |
| 2026-11-10 | fix-3 | deepseek-flash | no | 11972 | 4105 | 0 | 85760 | 0.0045 |
| 2026-11-10 | docs | deepseek-flash | no | 4287 | 7259 | 0 | 384 | 0.0050 |

Total: $0.0804. Prices: deepseek line, as measured by its adapter. Providers serve the current version of each model id, so the date tells which version answered; versions, launch stages and price sources per line are in the factory's model register (docs/MODELS.md).
