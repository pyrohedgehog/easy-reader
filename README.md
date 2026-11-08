# easy-reader

[![CI](https://github.com/pyrohedgehog/easy-reader/actions/workflows/ci.yml/badge.svg)](https://github.com/pyrohedgehog/easy-reader/actions/workflows/ci.yml)

A `go/analysis` linter for Go comment style. AI models have worsened a pet peeve of mine, and possibly many others: Documentation rules.

We've all read code that has far too much documentation, at one point this swung the other way, but this is not a solution to that.
My general ideal, being that, a method signature can convey most of the information, a short doc string should then make it easy to convey usage requirements not clear enough from the signature alone, and inside of the method, you convey points of complexity easier to miss. Different comment blocks for different needs, and each one easy to read.

Text that wraps excessively is
generally harder to read, I like to avoid
things like this, where you break your flow
and jot back and forth; especially when
slight shifts in word placement would completely
fix it!

To that end, these are the rules I personally have been trying to follow, to make my own life easier, when reading code.

## Checks

| Rule          | What it flags                                                                                                                                                                                                                                                                     |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| line-width    | Comment text over `max-line-width` characters. Exceptions automatically handled.                                                                                                                                                                                                  |
| punctuation   | Standalone comment lines not ending in `end-punctuation`. Trailing comments exempt.                                                                                                                                                                                               |
| block-length  | Comment blocks over `max-block-lines` lines (default 2), or `max-block-lines-private` on unexported declarations, including their bodies. A negative limit disables the check. Per-kind overrides: `package-doc`, `func-doc`, `type-doc`, `field-doc`, `value-doc`, `import-doc`. |
| strange-char  | Ambiguous/invisible Unicode in comments: em/en dashes, smart quotes, NBSP, ZWSP, BOM, etc.                                                                                                                                                                                        |
| fill *(TODO)* | A non-boundary comment line with room for the next line's first word. **Not yet implemented.**                                                                                                                                                                                    |
| empty-line    | A blank `//` line between non-empty comment lines (doc comments exempt, it is a paragraph break there).                                                                                                                                                                           |

### Exemptions:
- The first standalone comment run in a file (doc comments
don't consume it). This exempts the run from block-length only;
punctuation and line-width still apply to it.
- Doc comments attached to declarations (func/type/var/const/fields/specs, exempt from empty-line only).
- Runs containing an exemption marker. Block-length only; the markers
do not suppress the other checks.
- Generated files (`Code generated ... DO NOT EDIT.` before the package clause, per go.dev/s/generatedcode).

## Fixes

Reports carry suggested fixes applied by `golangci-lint run --fix`:

- **punctuation** -> appends the first `end-punctuation` character (default `.`) to the line, so the fixed line satisfies whatever `end-punctuation` setting is in effect. `/* */` comment lines are reported but not auto-fixed yet (see TODO).
- **line-width, block-length, empty-line** -> **reflow**: joins the run's paragraph text and re-wraps it at `max-line-width`. Boundary lines (directives, list items, code fences, markers, `{`/`}` endings) are preserved verbatim; blank `//` lines are collapsed outside doc comments. Only applies to pure `//` runs. Known issue: the reflow also collapses leading/trailing blank `//` lines, which the empty-line rule does not report (see TODO).

## Usage

### golangci-lint (module plugin)

In the consuming repo, `.custom-gcl.yml`:

```yaml
version: v2.13.2  # golangci-lint version to build
plugins:
  - module: github.com/pyrohedgehog/easy-reader
    import: github.com/pyrohedgehog/easy-reader/plugin   # package that registers the linter
    path: /path/to/easy-reader   # or `version: vX.Y.Z` once published
```

```sh
golangci-lint custom   # produces ./custom-gcl
./custom-gcl run
```

`.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - easyreader
  settings:
    custom:
      easyreader:
        type: module
        description: comment style rules
        settings:
          # Max comment length in characters.
          max-line-width: 150
          # Max consecutive inline comment lines on exported declarations.
          max-block-lines: 2
          # Max inline comment lines on unexported declarations, including their bodies (inherits max-block-lines).
          max-block-lines-private: 2
          # Allowed terminal characters for a standalone comment line.
          # The punctuation auto-fix appends the first one, so list the preferred terminator first.
          end-punctuation: ".,!?:"
          # Case-insensitive substrings exempting a run from block-length.
          exemption-markers: ["comment-length-exception", "nolint:commentlength", "nolint:cle", "lint:ignore"]
          # Lint files carrying the Code generated ... DO NOT EDIT. marker.
          check-generated: false
          # Optional per-kind block-length overrides (each takes max-block-lines and max-block-lines-private; 
          # zero inherits the globals above, a negative value disables the check for that kind).
          # Unexported resolution cascades: kind-private, kind-public,  global-private, global-public. 
          # Package docs are always exported.
          # Example: package docs unlimited, exported func docs 3 lines,
          # unexported 2:
          # package-doc:
          #   max-block-lines: -1
          #   max-block-lines-private: -1
          # func-doc:
          #   max-block-lines: 3
          #   max-block-lines-private: 2
```

### This repository

The repo lints itself with that setup: `.custom-gcl.yml` registers the plugin from the local module and `.golangci.yml` enables it, so the linter dogfoods its own rules. `pkg/testdata` is excluded - the analysistest fixtures contain deliberate violations.

```sh
make lint      # builds ./custom-gcl once, then runs it
make lint-fix  # same, with suggested fixes applied
make fmt       # gofmt + goimports
```

### go vet

```sh
go install github.com/pyrohedgehog/easy-reader/cmd/easy-reader@latest
go vet -vettool=$(which easy-reader) ./...
# Every setting except the per-kind doc limits is also an analyzer flag:
go vet -vettool=$(which easy-reader) -easyreader.max-line-width=200 ./...
```

`//nolint:` is honored only under golangci-lint; under `go vet` use the exemption marker.

## Layout

```
pkg/            - the easyreader package (import path github.com/pyrohedgehog/easy-reader/pkg)
  analyzer.go   - analysis.Analyzer wiring, per-file run
  config.go     - Settings + defaults + golangci settings mapstructure tags
  rules.go      - the checks
  comments.go   - comment-group -> per-line expansion, doc/generated/exemption helpers
  testdata/     - analysistest fixtures (`// want` annotations)
plugin/         - golangci-lint module-plugin registration
cmd/easy-reader - unitchecker binary for `go vet -vettool` (folder name = binary name)
```

## Test

```sh
go test ./...
make lint
make fuzz   # 30s per fuzz target by default; make fuzz FUZZTIME=5m for longer
```

# Future Items:
- [ ] settings for test file allowances.
- [ ] get fix working with long lines, auto breaking at punctuation when possible.
- [ ] fill rule, fully setup to reformat paragraphs to fit nicer.
- [ ] add min lines options
- [ ] dedupe identical suggested fixes: several diagnostics in one block can carry the same reflow edit.
- [ ] empty-line fix collapses leading/trailing blank lines the rule does not report; align fix with diagnostics.
- [ ] test coverage for the punctuation/boundary exemption edge cases: URLs, code fences, list items, separators, structural endings.
- [ ] make the exportedness inheritance of non-doc comments configurable (comments.go).
- [ ] line-width: base the limit on the comment text itself, ignoring leading indentation (rules.go).
- [ ] punctuation auto-fix for `/* */` comment lines (reported without a fix today).
- [ ] scheduled fuzz job: run `make fuzz` with a long FUZZTIME on a weekly cron.
- [ ] pkg: make the commentMetas AST walk clearer (comments.go).
- [ ] pkg: make reflow clearer (reflow.go).
- [ ] pkg: consider an interface for the exportedness helpers (comments.go).
- [ ] change over to using github issues entirely.
