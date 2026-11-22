# Code Navigation Graph

## Modules
- `main.go`: Thin entry point; wires `slog` stderr logger and runs `cmd.NewRootCommand()`.
- `cmd/root.go`: Cobra root command (`json2tsp [file]`), flags (`-n/--name`, `--inline`), file/stdin input handling, stdout output.
- `convert/`: Pure JSON-to-TypeSpec conversion pipeline (no CLI or filesystem dependencies):
  - `convert/parse.go`: Streaming `json.Decoder` parser (`UseNumber()`) preserving object key order.
  - `convert/infer.go`: Type inference (`int64`/`float64`, `plainDate`/`utcDateTime`, array merging, optional `?`, union widening).
  - `convert/emit.go`: TypeSpec document renderer (`ConvertJSONToTypeSpec`, named model extraction/deduplication, `--inline` mode, backtick identifier escaping).
- `build.sh`: Cross-compilation build script (`darwin/arm64`, `linux/amd64`).
- `web/index.html`: Standalone zero-dependency browser UI with parity JSON-to-TypeSpec conversion (ordered JSON parser, `int64`/`float64`, temporal scalars, array merging, `--inline`, file drag/drop, copy/download).

## Dependency Direction
`main.go` -> `cmd` -> `convert`
