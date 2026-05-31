# Development

## Layout

```
cmd/distill/            # CLI entry point + subcommand dispatch
internal/convert/       # the engine: Converter interface, Registry,
                        # priority dispatch, normalize() pipeline,
                        # StreamInfo, Result, type detection
internal/converters/
  src/                  # one file per format converter (package converters)
  tests/                # black-box tests (package converters_test)
internal/app/           # app-level features that compose the engine:
                        #   batch.go    — concurrent dir conversion
                        #   serve.go    — HTTP server
                        #   mcp.go      — MCP stdio server
docs/                   # user docs (this directory)
examples/               # ready-to-use samples (agent configs, etc.)
```

## Architecture

```
                 ┌───────────────┐
   bytes  ───►   │   Registry    │  ────►  Result { Markdown, Title, ...}
                 │  • priority   │
                 │  • Converter  │
                 │    interface  │
                 └──────┬────────┘
                        │ best-priority converter that Accepts()
                        ▼
            CSV / JSON / HTML / DOCX / PPTX / ODF /
            XLSX / EML / EPUB / Image / PDF / Media / ...
                        │
                        ▼
                  normalize()  ←  bullet-glyph + space + long-line
                                   wrap (skips fenced code)
```

- **Converters self-register** in `internal/converters/src/register.go`.
  Each one implements `Accepts(StreamInfo) bool` + `Convert(io.Reader,
  StreamInfo) (Result, error)`.
- **Lower priority value = tried first.** Specific formats (CSV, DOCX, PDF)
  use 0; the plain-text catch-all uses 10. PDFium overrides the default PDF
  converter at -1 when built with `-tags pdfium`.
- **`normalize()` runs on every output** so format-agnostic Markdown cleanup
  (bullet glyphs, NBSP, long-line wrap) lives in one place.

## Build matrix

```bash
go build              ./cmd/distill          # default,    pure-Go, ~9 MB
go build -tags pdfium ./cmd/distill          # full,       pure-Go, ~23 MB
```

There's no cgo build. `wazero` runs PDFium's WebAssembly module in pure Go.

## Tests

```bash
go vet ./...
go vet -tags pdfium ./...
go test ./... -race -count=1
go test -tags pdfium ./... -count=1

# Source coverage (tests are black-box, in a sibling package)
go test ./internal/converters/tests/ -coverpkg=./internal/converters/src
```

Slow tests (whisper transcription) are gated behind env vars:

```bash
DISTILL_TEST_WHISPER=1 go test ./internal/converters/tests/ -run Whisper -count=1
```

## Adding a new converter

1. Create `internal/converters/src/myformat.go`:

   ```go
   package converters

   import (
       "io"
       "github.com/MitudruDutta/distill/internal/convert"
   )

   type MyFormat struct{}

   func (MyFormat) Accepts(info convert.StreamInfo) bool {
       return info.Extension == ".myfmt"
   }
   func (MyFormat) Convert(r io.Reader, info convert.StreamInfo) (convert.Result, error) {
       // ... return convert.Result{Markdown: "..."}
   }
   ```

2. Register it in `internal/converters/src/register.go`:

   ```go
   reg.Register(MyFormat{}, 0)  // 0 = specific format
   ```

3. Add a black-box test in `internal/converters/tests/myformat_test.go`
   using `package converters_test` and dot-importing the source package.
4. Add a row to [`docs/formats.md`](formats.md).

The engine's `normalize()` will automatically clean up bullet glyphs / spaces
/ long lines in your output — you don't need to handle that.

## Style

- No external dep without a clear payoff. Stdlib first.
- Keep converters small; share helpers (`ziputil`, `external`, etc.).
- Tests are black-box (drive the public API); avoid testing internals when an
  external assertion is expressive enough.
- `gofmt` everything; CI runs `go vet` on both build tags.

## Releasing

Tag a commit on `main`:

```bash
git tag -a v0.1.1 -m "v0.1.1"
git push origin v0.1.1
```

The release workflow (`.github/workflows/release.yml`) runs GoReleaser to
build and publish cross-platform binaries to GitHub Releases. Snapshots from
non-tag pushes are not published.

## License

MIT — see [`LICENSE`](../LICENSE).
