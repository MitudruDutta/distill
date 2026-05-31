<h1 align="center">distill</h1>

<p align="center">
  <strong>Any document → clean Markdown, in a single Go binary, built for LLMs.</strong>
</p>

<p align="center">
  <a href="https://github.com/MitudruDutta/distill/actions/workflows/ci.yml"><img src="https://github.com/MitudruDutta/distill/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/MitudruDutta/distill/releases"><img src="https://img.shields.io/github/v/release/MitudruDutta/distill?display_name=tag&sort=semver" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/MitudruDutta/distill"><img src="https://pkg.go.dev/badge/github.com/MitudruDutta/distill.svg" alt="Go Reference"></a>
</p>

---

## Install — one command

```bash
curl -fsSL https://raw.githubusercontent.com/MitudruDutta/distill/main/install.sh | bash
```

That's it. The script detects your OS/arch, fetches the latest release, drops the binary in `~/.local/bin/distill`, and prints the next steps. It falls back to `go install` if a prebuilt release isn't available.

Other install paths: [`go install`, container, from source](docs/install.md).

---

## What it does

distill converts essentially anything you'd hand to an LLM — PDFs, Office documents, spreadsheets, presentations, e-books, e-mail, archives, images, audio, video — into clean, structured Markdown that LLMs were actually trained on. **No interpreter, no runtime, no model download.** Just one binary that runs anywhere.

```text
report.docx    ─┐
slides.pptx    ─┤
sheet.xlsx     ─┤      ┌──────────┐      # heading + ## sub
scan.pdf       ─┤  ──► │ distill  │ ──►  - bullet  - bullet
audio.mp3      ─┤      └──────────┘      | a | b | (real Markdown tables)
photo.jpg      ─┤
archive.zip    ─┘
```

| Status | Formats |
|--------|---------|
| ✅ | text, CSV/TSV, JSON, YAML/TOML/INI, XML, RSS/Atom, Jupyter, HTML, **DOCX**, **PPTX**, **XLSX**, ODT/ODS/ODP, EML, ZIP/TAR, EPUB, **PDF**, images, audio/video |

Per-format quality notes and limitations: [`docs/formats.md`](docs/formats.md).

---

## How to use it

There are **four ways to use distill**, all from the same binary. Pick the one that matches your goal.

### 1 · As an MCP tool inside an LLM agent (the killer use case)

`distill mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io) server over stdio. Wire it into any MCP client (Claude Desktop, Kiro CLI, custom agents) and your LLM gets a `convert` tool that handles every format above.

```jsonc
// ~/.kiro/settings/mcp.json   (or ~/.config/Claude/claude_desktop_config.json)
{
  "mcpServers": {
    "distill": {
      "command": "/home/USER/.local/bin/distill",
      "args": ["mcp"]
    }
  }
}
```

**Make the agent always convert first** — when a user pastes a PDF/DOCX/etc. path, the LLM should call `@distill/convert` *before* trying to reason. Three patterns (system prompt, dedicated agent config, deterministic preprocessing hook) with copy-paste examples: [`docs/agents.md`](docs/agents.md). A drop-in Kiro CLI agent ships at [`examples/agents/distill-aware.json`](examples/agents/distill-aware.json).

### 2 · As a CLI

```bash
distill report.docx                    # → Markdown to stdout
distill report.docx -o report.md       # → write to a file
cat data.csv | distill -x csv          # → stdin
distill report.pdf -json               # → structured JSON model
distill https://example.com/paper.pdf  # → fetch a URL (SSRF-safe by default)
distill 'data:text/csv;base64,YSxiCjEsMgo='  # → data: URI
distill file:///etc/hosts              # → file: URI
```

Flags: `-o` output · `-x` extension hint · `-m` MIME hint · `-c` charset · `-json`. Full reference: [`docs/usage.md`](docs/usage.md).

### 3 · As a concurrent batch converter

```bash
distill batch ./corpus --out-dir ./md           # parallel dir-tree conversion
distill batch ./corpus --out-dir ./json --json  # JSON sidecars
distill batch ./corpus --out-dir ./md --workers 16
```

Workers default to `GOMAXPROCS`. Output mirrors input layout. Continue-on-error with a final summary. Built for ingesting an entire corpus into a RAG pipeline in one shot.

### 4 · As an HTTP service

```bash
distill serve --addr 127.0.0.1:8080                  # secure-by-default
distill serve --addr 0.0.0.0:9090 --token "$TOKEN"   # exposed; auth required
```

```bash
curl --data-binary @file.pdf http://127.0.0.1:8080/convert
curl --data-binary @data.csv "http://127.0.0.1:8080/convert?ext=csv&format=json"
```

`POST /convert` (raw body) · `GET /healthz`. **Refuses to bind a non-loopback address without a token.** Body cap, request timeouts, constant-time auth compare. Details: [`docs/serve.md`](docs/serve.md).

---

## Why distill produces LLM-quality Markdown

Every converter shares one engine pass that:

- normalizes Unicode bullet glyphs (`•`, `▪`, `●`, plus the **Private Use Area** that most resume/CV fonts hide their bullets in) → `- ` with indent preserved;
- folds exotic Unicode whitespace (NBSP, em/en/ideographic) to ASCII space and drops zero-width / BOM codepoints (LLM token efficiency);
- splits implausibly long paragraphs at sentence boundaries (so a 1 MB single-line `<p>` becomes one sentence per line — perfect for chunking);
- preserves fenced code blocks **byte-for-byte** so JSON/YAML/code stays exact.

For PDFs (with `-tags pdfium`), it goes further: **font-size heading detection**, bullet-glyph recognition, and **layout-based table reconstruction** — the resume goes from a flat text dump to `# Name → ### Key Skills → - bullet, - bullet`.

```markdown
## Slide 2: Fruit Data

| Fruit   | Quantity | Price per Unit |
| ---     | ---      | ---            |
| Apples  | 5        | $0.50          |
| Mangoes | 3        | $1.00          |
```

That's what your LLM sees. Without distill, the same input is binary OOXML.

---

## Build tiers

```bash
go build              ./cmd/distill   # default: pure-Go, ~9 MB
go build -tags pdfium ./cmd/distill   # full: + PDFium WASM, ~23 MB
```

The `pdfium` build embeds PDFium as **WebAssembly via [wazero](https://wazero.io)** — pure-Go, no cgo, no system library. It unlocks the high-fidelity PDF engine.

## Optional external tools (auto-detected)

distill stays self-contained. If these tools are present on `PATH`, additional features light up automatically; if absent, distill degrades gracefully.

| Tool | Unlocks |
|------|---------|
| `pdftotext` (poppler) | higher-fidelity PDF text |
| `pdftoppm` (poppler) | rasterizing scanned PDFs for OCR |
| `tesseract` | image OCR + scanned-PDF OCR |
| `ffmpeg` / `ffprobe` | audio/video metadata |
| `whisper` (`openai-whisper`) | speech transcription |

---

## Documentation

| Topic | Link |
|-------|------|
| Install (one-liner, `go install`, container, source) | [`docs/install.md`](docs/install.md) |
| CLI usage and all four modes | [`docs/usage.md`](docs/usage.md) |
| Per-format quality notes | [`docs/formats.md`](docs/formats.md) |
| MCP server protocol & client setup | [`docs/mcp.md`](docs/mcp.md) |
| **Agent integration patterns** (auto-convert before LLM) | [`docs/agents.md`](docs/agents.md) |
| HTTP API contract | [`docs/serve.md`](docs/serve.md) |
| **Plugin protocol** (add a format without recompiling) | [`docs/plugins.md`](docs/plugins.md) |
| Threat model & hardening | [`docs/security.md`](docs/security.md) |
| Architecture & adding a converter | [`docs/development.md`](docs/development.md) |

---

## Architecture in one paragraph

A small **engine** (`internal/convert`) holds a priority registry of **converters**. Each declares which streams it accepts and how to turn them into a `Result{Markdown, Title, Headings, Tables}`. The engine runs every output through one shared `normalize()` pass (bullet glyphs, exotic spaces, long-line wrap, code-fence preservation). Format-specific code only handles what's truly format-specific — PDFium font sizes, OOXML heading styles, OPF spine order, and so on.

## Contributing

PRs welcome. Run `make test-race` before opening one. Adding a format takes about 50 lines of Go plus a black-box test — the [development guide](docs/development.md) walks through it.

## License

[MIT](LICENSE)
