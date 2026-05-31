# Install

distill ships as a single, statically-linked binary. There is no runtime,
interpreter, or model to download for the default build.

## One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/MitudruDutta/distill/main/install.sh | bash
```

The script:

- detects your OS (Linux/macOS) and architecture (amd64/arm64),
- downloads the latest release from GitHub,
- installs to `~/.local/bin/distill` (override with `DISTILL_PREFIX`),
- falls back to `go install` if a release isn't available for your platform.

Pin a version with `DISTILL_VERSION=v0.1.1`.

## With Go

```bash
go install github.com/MitudruDutta/distill/cmd/distill@latest
```

This always works wherever Go ≥ 1.22 is installed and gives you the **default
build** (no PDFium engine; ~9 MB).

For the **full-feature build** (PDFium WebAssembly engine for higher-fidelity
PDF text + table reconstruction; ~23 MB):

```bash
git clone https://github.com/MitudruDutta/distill && cd distill
go build -tags pdfium -o ~/.local/bin/distill ./cmd/distill
```

## Container

```bash
docker run --rm -v "$PWD:/work" ghcr.io/mitudrudutta/distill:latest /work/file.pdf
```

The image is distroless and runs as `nonroot`. Mount your input directory and
pass paths under that mount.

## From source

```bash
git clone https://github.com/MitudruDutta/distill
cd distill
make build         # default
make build-pdfium  # full-feature
make install       # → ~/.local/bin/distill
```

## Optional external tools (auto-detected at runtime)

The default binary is fully self-contained. These tools, if present on `PATH`,
**unlock additional features**:

| Tool | Unlocks |
|------|---------|
| `pdftotext` (poppler) | higher-fidelity PDF text |
| `pdftoppm` (poppler) | rasterizing scanned PDFs for OCR |
| `tesseract` | image OCR + scanned-PDF OCR |
| `ffmpeg` / `ffprobe` | audio/video metadata |
| `whisper` (`openai-whisper`) | speech transcription on audio/video |

Install whatever you need. distill **degrades gracefully** when a tool is
missing — it just won't activate that feature.

## Verify the install

```bash
distill --help
echo 'a,b
1,2' | distill -x csv          # expect a Markdown table
distill ~/some.pdf -o some.md  # convert a real file
```

## Uninstall

```bash
rm ~/.local/bin/distill
```

If you wired it into an MCP client, also remove the entry from the client's
config (e.g. `~/.kiro/settings/mcp.json`).
