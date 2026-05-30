# Supported formats

All converters share a generic Markdown post-processing pass that:

- normalizes Unicode bullet glyphs (`•`, `▪`, `●`, …, plus the Private Use
  Area used by many resume/CV fonts) to `- ` while preserving indent,
- folds exotic Unicode whitespace (NBSP, em/en/ideographic spaces, …) to
  ASCII space and drops zero-width / BOM codepoints (LLM token efficiency),
- wraps implausibly long paragraphs at sentence boundaries (so a single
  10,000-char `<p>` becomes one sentence per line for clean RAG chunking),
- preserves fenced code blocks **byte-for-byte** so JSON / YAML / source
  inside ` ``` ` blocks is never altered.

| Format | Quality | How |
|--------|---------|-----|
| Plain text | passthrough | stdlib |
| CSV / TSV | aligned Markdown table | stdlib `encoding/csv` |
| JSON | pretty-printed `json` fence | stdlib `encoding/json` |
| YAML / TOML / INI | language-tagged fence (faithful) | stdlib |
| XML | `xml` fence; RSS/Atom feeds get a title heading + dated link list | stdlib |
| Jupyter `.ipynb` | markdown cells verbatim, code cells fenced with the language, text outputs included | stdlib |
| HTML | `# / ## / ###`, links, lists, code, blockquotes | [html-to-markdown/v2](https://github.com/JohannesKaufmann/html-to-markdown) |
| DOCX | Heading1..N → `#..######`, `numPr` → bullet lists with indent, `w:tbl` → Markdown tables | stdlib zip+xml |
| PPTX | `## Slide N: <title>` (title placeholder detection), `a:pPr lvl` → bullet indent, `a:tbl` in graphic frames → tables | stdlib zip+xml |
| XLSX | each sheet → `## <name>` + Markdown table | [xuri/excelize/v2](https://github.com/qax-os/excelize) |
| ODT / ODS / ODP | `text:outline-level` → headings, nested `text:list` → indented bullets, `table:table` → tables | stdlib zip+xml |
| EML | RFC 822 headers (decoded MIME words) + multipart walk; HTML parts converted via the HTML pipeline; base64/quoted-printable decoded | stdlib `net/mail` |
| ZIP / TAR | iterates entries, converts each via the registry (recursion bound; per-entry/total byte caps; nested archives skipped) | stdlib |
| EPUB | resolves OPF spine order; converts each XHTML chapter via the HTML pipeline | stdlib + html-to-markdown |
| Images | format + pixel dimensions; OCR text appended when `tesseract` is on `PATH` | stdlib `image.DecodeConfig` + tesseract |
| PDF | `pdftotext -layout` if available → pure-Go fallback → OCR fallback. With `-tags pdfium`: PDFium WASM engine with **font-size heading detection**, bullet-glyph (incl. PUA) recognition, and **layout-based table reconstruction** | poppler / `ledongthuc/pdf` / `klippa-app/go-pdfium` |
| Audio / Video | `ffprobe` metadata block; transcript appended when `whisper` is on `PATH` | ffmpeg + openai-whisper |

## Honest limitations

- **PDF kerning artifacts**: source PDFs often render words with letter spacing
  (`S c i k i t   - l e a r n`). The text extractor sees those as separate
  tokens (`Scikit - learn`). Fixing this generically is risky (a real
  hyphenated word like `data - driven` looks identical), so distill leaves it.
- **PDF without semantic structure**: most PDFs have none. The PDFium font-size
  heuristic recovers section headings when the source uses a clearly larger
  font; otherwise output is paragraphs (still LLM-readable).
- **DOCX/PPTX without heading styles**: if a document uses bold-paragraph
  formatting instead of real Heading styles, those won't be recovered as `#`.
- **OCR / transcription quality**: depends on input quality and the
  installed model (Tesseract languages, Whisper model size). distill calls
  the tool as-is — tune the tool, not distill.

## Build tiers

The default binary is small and pure-Go. The `pdfium` build tag adds
~14 MB and unlocks the high-fidelity PDF engine (still no cgo — uses
`wazero` to run PDFium's WebAssembly module).

```bash
go build -o distill ./cmd/distill                  # default, ~9 MB
go build -tags pdfium -o distill ./cmd/distill     # full, ~23 MB
```

The PDFium engine is **lazy-initialized**: the first PDF call pays a ~1 s WASM
startup; subsequent calls in the same process are fast.
