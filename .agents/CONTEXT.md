# Project Context

## Overview
`json2tsp` is a Go CLI tool that reads JSON from a file or `stdin` and prints corresponding TypeSpec definitions to `stdout`.

## Tech Stack
- Language: Go 1.27 (`go.mod`)
- CLI Framework: `github.com/spf13/cobra`
- Logging: `log/slog` (errors to `stderr`)

## Type Mapping Conventions
- Nested JSON objects: extracted into named `model` declarations by default; `--inline` switches to inline anonymous object types.
- Numbers: integers -> `int64`, decimals/scientific notation/overflow -> `float64` (`int64` + `float64` widens to `float64`).
- Strings: `YYYY-MM-DD` -> `plainDate`, `HH:MM:SS(.fff)` -> `plainTime`, RFC 3339 with `Z` -> `utcDateTime`, RFC 3339 with `±HH:MM` -> `offsetDateTime` (`utcDateTime` + `offsetDateTime` widens to `offsetDateTime`), others -> `string` (temporal + `string` widens to `string`).
- Object arrays: fields are merged across elements; fields absent from any element are marked optional (`?`).
- Identifiers: original JSON key order is preserved; invalid TypeSpec identifiers or reserved keywords are wrapped in backticks.

## Verification Commands
- Vet: `go vet ./...`
- Test: `go test -race ./...`
- Build: `bash build.sh`
