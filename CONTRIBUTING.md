# Contributing

## Setup

- Go 1.27, matching go.mod; the Makefile also pins `GOTOOLCHAIN` for the lint build.
- `make test`  run the test suite.
- `make lint`  build `./custom-gcl` (once), then lint, including this repo's own comment rules.
- `make fuzz`  short fuzz session per target; raise with `FUZZTIME=5m`.

## The comment formats

This repo lints itself: every Go comment must satisfy easyreader's own rules, as their default state.
(max 2 lines per block, terminal punctuation, 150 char width). If linting fails, either the comment is outside default spec, or ideally, you are able to raise an issue :).

`make lint` only rebuilds `./custom-gcl` when `.custom-gcl.yml` changes; if
`pkg/` or `plugin/` changed since it was built, remove the binary and re-run
(make prints a warning when this applies).

## Testing conventions

- Analyzer behavior is tested with analysistest fixtures under `pkg/testdata/src/<pkg>`, using `// want` annotations next to the offending line.
- Unit tests use testify (`assert`/`require`).
- Fuzz targets live in `pkg/fuzz_test.go`; run `make fuzz` after touching comment parsing or fix generation.

## Pull requests

- CI runs `go test ./...` and `make lint` on every push; both must pass.
- Reference github issues please.
