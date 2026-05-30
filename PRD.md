# PRD — `distill`: a complete document → Markdown engine in Go

> Working name: **distill** (binary `distill`). Rename freely — it lives in one place:
> the `module` line in `go.mod`.
>
> Status: Draft v1 · Date: 2026-05-30 · Owner: <you> · Language: Go

---

## 1. Problem statement

LLM and RAG pipelines need clean, lightly-structured Markdown extracted from *any* document —
PDF, Office, OpenDocument, e-mail, archives, e-books, images, audio, video, and web pages.
Tools in this space are typically Python with many heavy dependencies (data-frame libraries,
ML models, cloud SDKs), giving slow cold starts, large footprints, dependency conflicts, and
no clean single-binary deployment.

`distill` is a ground-up product that converts essentially **any** document to Markdown,
shipped as a **single Go binary** with optional capabilities layered behind build tags.
It is optimized for **high-throughput batch conversion** and **machine-consumable structured
output**, while still aiming for **broad format coverage and good fidelity**.

## 2. Vision — what "full package" means

A user can point `distill` at *anything* and get useful Markdown:

- Every common text, markup, Office, OpenDocument, e-mail, archive, and e-book format.
- **PDF** with real text extraction **and table reconstruction**, with **OCR fallback** for
  scanned pages.
- **Images** → metadata + **OCR** text + optional **LLM caption**.
- **Audio/Video** → metadata + **speech transcription**.
- **Web** → fetch a URL (SSRF-safe) and convert; first-class handling for common sources.
- Optional **cloud document-AI** and **ML layout** backends as pluggable, opt-in providers.
- Surfaces: one-shot CLI, concurrent **batch**, **server** (`POST /convert`), an **MCP**
  server for agent tooling, and a **plugin** system for third-party converters.
- Every conversion can also emit a **structured JSON sidecar** (title, headings tree,
  tables, detected type, metadata).

Heavy capabilities (OCR, transcription, PDFium, cloud, ML) are **opt-in** via build tags or
runtime flags, so the default binary stays small and pure-Go.

## 3. Goals / Non-goals

**Goals**
- G1. Single statically-linked default binary; heavy features behind build tags.
- G2. **Broad offline coverage**: text, CSV/TSV, JSON, YAML, INI/TOML, XML/RSS/Atom,
  Jupyter (`.ipynb`), HTML, DOCX/XLSX/XLS/PPTX, ODT/ODS/ODP, EML/MSG, ZIP/TAR, EPUB,
  image metadata.
- G3. **PDF**: text extraction, table reconstruction, OCR fallback for scanned pages.
- G4. **OCR** for images and scanned PDFs (Tesseract-backed, build-tag gated).
- G5. **Audio/Video transcription** (local engine or cloud provider, runtime-selectable).
- G6. **Web ingestion**: SSRF-safe URL fetch; convert remote documents and pages.
- G7. **Optional intelligence**: LLM image captioning; pluggable cloud document-AI and ML
  layout backends.
- G8. **Concurrent batch mode** across all cores — headline performance feature.
- G9. **Structured JSON sidecar** alongside Markdown.
- G10. Product surfaces: CLI, `batch`, `serve` (secured), `mcp`, and a plugin API.
- G11. Build + unit tests green at every phase; reproducible cross-compilation.

**Non-goals**
- N1. Not a document **editor**, viewer, or renderer (we output Markdown, not render to it).
- N2. Not a format-to-format converter beyond Markdown (no DOCX→PDF, etc.).
- N3. Not a translation or summarization service (we *extract*; we do not rewrite prose).
- N4. Not a general-purpose web crawler (we fetch the documents you name, not the web).

## 4. Target users

- RAG / LLM engineers ingesting large, heterogeneous document corpora.
- Backend teams wanting a small-container `POST /convert` service.
- Agent builders needing an MCP tool for "read this file as Markdown."
- CLI / automation users who want `find . | distill` to "just work" — no virtualenv.

## 5. Market context (honest positioning)

The category spans Python ML-layout tools (high fidelity, heavy), JVM content extractors
(mature, large runtime), and broad general converters (not LLM-output-shaped). The unowned
intersection is **zero-dependency single-binary distribution + concurrent batch + structured
output + agent-native (MCP) surface**. That intersection is the wedge — coverage and speed
are table stakes we also intend to meet.

## 6. Differentiation (the wedge)

1. **Single binary, layered capabilities** — pure-Go core; OCR/PDFium/transcription opt-in.
2. **Concurrent batch** — saturate all cores over tens of thousands of files.
3. **Structured JSON sidecar** — a parseable document model for RAG chunking.
4. **Agent-native** — first-class MCP server, not an afterthought.
5. **Secure-by-default server** — locked down out of the box (§10).

## 7. Language & build strategy

**Go**, for fast iteration, concurrency, cross-compilation, and small static binaries.

Capability tiers via **build tags** so the default build is tiny and dependency-free:

- `core` (default): all stdlib-only converters (text family, OOXML/ODF via zip+xml, archives).
- `html`, `xlsx`: pure-Go third-party libs (small).
- `pdf` (cgo): PDFium-backed extraction + tables.
- `ocr` (cgo): Tesseract.
- `media`: audio/video transcription backend.
- `cloud`: optional document-AI / LLM providers.

A `full` tag composes them for the "everything" binary.

## 8. Format scope & conversion strategy

> Dependency rule: **before** adding any dependency at a phase, confirm it exists and is
> maintained (`go get` + read its README). Never trust a package name from memory.
> Status legend: ✅ verified 2026-05-30 · 🔎 verify at phase start.

| Format | Strategy | Dependency / mechanism | Tier |
|--------|----------|------------------------|------|
| Plain text | passthrough + charset normalize | stdlib | core |
| CSV / TSV | parse → Markdown table | stdlib `encoding/csv` | core |
| JSON | pretty / fenced / flatten | stdlib `encoding/json` | core |
| YAML / TOML / INI | parse → fenced / key-value | 🔎 small pure-Go parsers | core |
| XML / RSS / Atom | parse → headings / lists | stdlib `encoding/xml` | core |
| Jupyter `.ipynb` | JSON → cells → Markdown | stdlib `encoding/json` | core |
| HTML | DOM → Markdown | `JohannesKaufmann/html-to-markdown/v2` ✅ | html |
| DOCX | unzip → `word/document.xml` → MD | stdlib `archive/zip` + `encoding/xml` | core |
| PPTX | unzip → slide XML → MD | stdlib `archive/zip` + `encoding/xml` | core |
| XLSX / XLS | sheets → Markdown tables | `github.com/xuri/excelize/v2` ✅ | xlsx |
| ODT / ODS / ODP | unzip → `content.xml` → MD | stdlib `archive/zip` + `encoding/xml` | core |
| EML | parse MIME → MD | stdlib `net/mail`, `mime` | core |
| MSG (Outlook) | parse OLE/CFB → MD | 🔎 OLE/CFB reader | core |
| ZIP / TAR | iterate entries, recurse | stdlib `archive/zip`,`archive/tar` | core |
| EPUB | unzip → reuse HTML pipeline | stdlib + html-to-markdown | html |
| Images (metadata) | EXIF / dimensions block | `github.com/rwcarlsen/goexif` 🔎 | core |
| Images (OCR) | Tesseract → text | `gosseract` / `tesseract` CLI 🔎 | ocr |
| Images (caption) | LLM vision → alt text | provider API (opt-in) | cloud |
| PDF (text) | extract text | `klippa-app/go-pdfium` ✅ (cgo); `ledongthuc/pdf`/`rsc.io/pdf` ✅ pure-Go fallback | pdf |
| PDF (tables) | layout/word-position heuristics | on top of PDFium word boxes | pdf |
| PDF (scanned) | rasterize → OCR | PDFium + Tesseract | pdf+ocr |
| Audio | metadata + transcription | tags lib + Whisper/cloud 🔎 | media |
| Video | metadata + audio transcription | ffmpeg demux + transcription 🔎 | media |
| URL fetch | SSRF-safe GET → dispatch | stdlib `net/http` + guards | core |

## 9. Engine architecture

Small core engine + independent, self-registering converters (strategy +
chain-of-responsibility):

- **`Converter` interface**: `Accepts(info) bool`, `Convert(stream, info) (Result, error)`.
- **`StreamInfo`**: mimetype, extension, charset, filename, localPath, url (all optional).
- **`Result`**: `Markdown`, `Title`, plus optional structured fields (headings, tables) for
  the JSON sidecar.
- **`Registry`**: ordered `(converter, priority)`; **lower priority value tried first**.
  Specific formats (priority 0) precede generic catch-alls (priority 10).
- **Dispatch loop**: build type guesses → for each guess, try each accepting converter in
  priority order → first success wins → normalize trailing whitespace and collapse blank
  lines.
- **Type detection**: extension hint → stdlib `http.DetectContentType` (magic bytes) →
  optional shell-out to a `magika` CLI if present on `PATH`.
- **Capability gating**: tier-specific converters register only when built with their tag;
  a missing capability yields a clear "built without X" error, never a silent wrong result.

```
go.mod                         # module github.com/MitudruDutta/distill
cmd/distill/main.go            # CLI: file|stdin -> stdout|-o; subcommands: batch, serve, mcp
internal/convert/
  streaminfo.go                # StreamInfo, Result
  registry.go                  # Converter interface, Registry, dispatch loop
  detect.go                    # extension + magic-byte detection
internal/converters/
  register.go                  # wire built-ins with priorities
  plaintext.go                 # generic catch-all (priority 10)
  csv.go                       # specific (priority 0)
  ...                          # html, docx, xlsx, pdf, ocr, media, ... per phase
internal/app/                  # batch, serve, mcp, sidecar wiring (later phases)
```

## 10. Security requirements (non-negotiable)

- I/O runs with process privileges; document this clearly.
- **`serve`**: bind `127.0.0.1` by default; require an auth token for any non-loopback bind;
  enforce request-size limits and per-request timeouts. Never ship an open, unauthenticated
  service by default.
- **URL fetching**: off unless requested; block private/loopback/link-local/cloud-metadata
  IP ranges (SSRF); allowlist schemes (`http`,`https`,`file`,`data`).
- **ZIP/EPUB/TAR**: guard zip-slip (path traversal), zip-bombs (entry-count/size caps),
  and recursion depth.
- **Cloud/LLM providers**: keys via env only; opt-in; never log payloads or secrets.
- Treat all document content as untrusted; never execute embedded macros/scripts.

## 11. Success metrics

- M1. Cold-start + convert a 1-page DOCX in **< 50 ms** (pure-Go build).
- M2. Batch-convert 10,000 mixed files across all cores with near-linear scaling.
- M3. Default (`core+html+xlsx`) binary **≤ 25 MB**; `full` binary documented separately.
- M4. ≥ 80% line coverage on the core engine and each pure-Go converter.
- M5. Byte-stable (deterministic) output for identical input on text formats.

## 12. Risks (honest)

- R1. **PDF tables & scanned PDFs** are the hardest, highest-effort area. Mitigation: ship
  text first, tables next, OCR last; gate behind `pdf`/`ocr` tags; be explicit about limits.
- R2. **OCR / transcription** require cgo or external binaries/models → bigger builds and
  platform friction. Mitigation: opt-in tiers; cloud alternative; clear install docs.
- R3. **Type detection** without a bundled ML model. Mitigation: extension + magic bytes
  cover the supported set; optional `magika` CLI passthrough.
- R4. **OOXML/ODF edge cases** are sprawling. Mitigation: incremental, real-file tests.
- R5. **Scope is large.** Mitigation: strict phase gates; each phase ships a working binary.

## 13. Milestones → phases (full breakdown in TASKS.md)

- **Phase 0** — Core engine + CLI + plain-text/CSV. Build & tests green. ← start here
- **Phase 1** — Text family: JSON, YAML/TOML/INI, XML/RSS/Atom, ipynb, HTML.
- **Phase 2** — Office/ODF/e-mail/archives/e-books: DOCX, PPTX, XLSX/XLS, ODT/ODS/ODP,
  EML/MSG, ZIP/TAR, EPUB, image metadata.
- **Phase 3** — PDF: pure-Go text → PDFium text → table reconstruction.
- **Phase 4** — OCR: images and scanned PDFs (Tesseract, `ocr` tag).
- **Phase 5** — Media: audio/video metadata + transcription (`media` tag, local or cloud).
- **Phase 6** — Web & intelligence: SSRF-safe URL fetch; optional LLM caption; optional
  cloud document-AI / ML-layout providers (`cloud` tag).
- **Phase 7** — Product surfaces: concurrent batch, JSON sidecar, secured `serve`, `mcp`
  server, plugin API.
- **Phase 8** — Packaging & release: build-tag matrix, cross-compile, container, Homebrew/
  Scoop, docs, honesty pass on fidelity limits.
