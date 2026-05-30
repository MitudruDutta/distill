# TASKS — `distill` build plan

Detailed, checkable breakdown mapped to `PRD.md`. Work top-to-bottom; each phase ends with a
green build + tests and a usable binary.

## Conventions

- `- [ ]` = todo, `- [x]` = done. Tasks are numbered `phase.task`.
- **Tier tags** (from PRD §7): `core` (default, stdlib-only), `html`, `xlsx`, `pdf` (cgo),
  `ocr` (cgo), `media`, `cloud`, `full`.
- **Every converter ships with**: `Accepts`, `Convert`, registry wiring, a table-driven test
  with a `testdata/` fixture, and a one-line entry in the README format matrix.
- **Dependency rule**: before using any third-party lib, run `go get` and read its README/godoc
  to confirm it exists and is maintained. Status: ✅ verified 2026-05-30 · 🔎 verify at phase.
- **Definition of Done (per phase)**: `go vet ./...`, `go build ./...` (and each relevant tag),
  `go test ./... -race` all pass; new formats produce expected Markdown on a real sample.

---

## Phase 0 — Core engine + CLI + text/CSV  ← START

Goal: a running binary that converts plain text and CSV, with the dispatch architecture in
place. Zero third-party dependencies.

- [ ] 0.1 `go mod init github.com/MitudruDutta/distill`; set Go 1.26; create dir layout from PRD §9.
- [ ] 0.2 `internal/convert/streaminfo.go`: `StreamInfo{Mimetype,Extension,Charset,Filename,LocalPath,URL}` + `Merge(other) StreamInfo`.
- [ ] 0.3 `internal/convert/streaminfo.go`: `Result{Markdown,Title,Headings,Tables}` (structured fields nullable, used later by the sidecar).
- [ ] 0.4 `internal/convert/registry.go`: `Converter` interface — `Accepts(StreamInfo) bool`, `Convert(io.Reader, StreamInfo) (Result, error)`.
- [ ] 0.5 `internal/convert/registry.go`: `Registry` with `Register(conv, priority)`, stable priority sort (low value first), `Convert(reader, guesses)` dispatch loop (first success wins).
- [ ] 0.6 Output normalization in dispatch: strip trailing whitespace per line; collapse 3+ blank lines to 1.
- [ ] 0.7 `internal/convert/detect.go`: build `[]StreamInfo` guesses from extension (`mime.TypeByExtension`), magic bytes (`http.DetectContentType`), and a charset sniff.
- [ ] 0.8 `internal/converters/plaintext.go`: `PlainTextConverter`, priority 10, accepts `text/*` and unknown/empty types.
- [ ] 0.9 `internal/converters/csv.go`: `CSVConverter`, priority 0, `.csv`/`.tsv` → aligned Markdown table (header + separator + rows), delimiter inferred from extension.
- [ ] 0.10 `internal/converters/register.go`: `Default() *Registry` wiring the built-ins.
- [ ] 0.11 `cmd/distill/main.go`: read file arg or stdin; flags `-o/--output`, `-x/--extension`, `-m/--mime-type`, `-c/--charset`; write stdout or file; non-zero exit on error.
- [ ] 0.12 Tests: registry priority/dispatch, CSV table rendering (incl. quoting/commas), plaintext passthrough, detection guesses.
- [ ] 0.13 Run `go vet`, `go build ./...`, `go test ./... -race`. **DoD**: `printf 'a,b\n1,2\n' | distill -x csv` prints a Markdown table.

---

## Phase 1 — Text family ✅ (done 2026-05-30)

Goal: cover structured-text formats. Mostly stdlib; one HTML dependency.

- [x] 1.1 `json.go`: validate + pretty-print into a fenced ```json block. (stdlib; `--flatten` deferred)
- [x] 1.2 `fenced.go`: YAML as a language-tagged fenced block (faithful zero-dep passthrough, chosen over lossy parse/re-serialize).
- [x] 1.3 `fenced.go`: TOML and INI rendered the same way (zero-dep). Parser-based key/value rendering can be added later.
- [x] 1.4 `fenced.go`: generic XML as a ```xml fenced block. (Structured headings/lists walk deferred.)
- [x] 1.5 `feed.go`: detect RSS/Atom; title + linked, dated item list; non-feeds error and fall through to the XML fence.
- [x] 1.6 `ipynb.go`: parse notebook JSON; markdown cells verbatim, code cells fenced with language, text outputs included.
- [x] 1.7 `html.go` via `github.com/JohannesKaufmann/html-to-markdown/v2` ✅ (`ConvertReader`). NOTE: in the default build for now; `html` build tag and `--keep-data-uris` deferred to Phase 8.
- [x] 1.8 Registered all with priorities (specific=0, markup/fence + HTML=5, plain-text catch-all=10); table-driven tests inline (testdata fixtures deferred).
- [x] 1.9 **DoD met**: `go vet`, `go build`, `go test -race` green; every format verified end-to-end via the CLI.

---

## Phase 2 — Office, OpenDocument, e-mail, archives, e-books, image metadata

Goal: the bulk of "office" coverage. OOXML/ODF are zip+xml (stdlib); XLSX uses excelize.
Phase 2A (text/office formats) is done; Phase 2B (EML, archives, EPUB) is next.

- [x] 2.1 `ziputil.go`: open a zip from an `io.Reader` (buffers it) and read a named entry. (Kept in-package, not a sub-package.)
- [x] 2.2 `docx.go` + shared `xmltext.go`: extract paragraph text from `word/document.xml`. NOTE: headings/tables/lists are flattened to paragraphs (text preserved); structured heading/table output deferred.
- [x] 2.3 `pptx.go`: iterate `ppt/slides/slideN.xml` in slide order; emit `## Slide N` + text. Title-vs-body distinction deferred.
- [x] 2.4 `xlsx.go` via `github.com/xuri/excelize/v2` ✅: each sheet → `## <name>` + Markdown table. NOTE: `.xlsx` only — legacy binary `.xls` deferred (excelize does not read BIFF).
- [x] 2.5 `odf.go`: ODT/ODS/ODP via `content.xml` — `text:h` and `text:p` text extracted. Tables flattened (deferred).
- [x] 2.6 `eml.go`: headers + MIME multipart (prefers text/plain, HTML parts converted); base64/quoted-printable decoded.
- [ ] 2.7 MSG (OLE/CFB) — deferred (needs `mscfb` + MAPI property parsing).
- [x] 2.8 `archive.go`: ZIP/TAR entries converted via the registry with entry-count/byte caps; nested archives skipped (recursion bound); no disk writes (zip-slip N/A).
- [x] 2.9 `epub.go`: resolve OPF via container.xml; convert spine XHTML in order through the HTML pipeline.
- [x] 2.10 `image.go`: format + pixel dimensions via stdlib `image.DecodeConfig` (png/jpeg/gif). NOTE: full EXIF table deferred (kept zero-dep, no `goexif`).
- [x] 2.11 Usage-named tests with programmatically-built fixtures (`docx_test.go`, `pptx_test.go`, `odf_test.go`, `xlsx_test.go`, `image_test.go`).
- [x] 2.12 **DoD met (Phase 2A)**: `go vet`/`build`/`test -race` green; converters coverage 87.6%.

---

## Phase 3 — PDF (text → PDFium → tables)

Goal: solid text extraction by default, high-fidelity + tables behind the `pdf` (cgo) tag.

- [x] 3.1 `pdf.go` (default, no cgo): pure-Go text via `github.com/ledongthuc/pdf` ✅ (`GetPlainText`), panic-hardened. Tested with an fpdf-generated PDF round-trip.
- NOTE (3B): higher-fidelity text now uses poppler's `pdftotext -layout` when present (no cgo); scanned PDFs fall back to OCR (`pdftoppm` + `tesseract`). PDFium bindings and true Markdown **table reconstruction** remain the optional cgo route (deferred).
- [ ] 3.2 Verify + add `github.com/klippa-app/go-pdfium` ✅; `pdf_pdfium.go` (`//go:build pdf`): init instance pool, open from bytes, extract text per page, close pages to bound memory.
- [ ] 3.3 Build-tag dispatch: `pdf` tag overrides pure-Go converter at registration.
- [ ] 3.4 Word-box extraction from PDFium (position + size per word) for layout work.
- [ ] 3.5 Table reconstruction: cluster words into rows (Y) and columns (X) with adaptive tolerance; render aligned Markdown tables; pass through non-table text as paragraphs.
- [ ] 3.6 Fallback chain: PDFium fails/empty → pure-Go path; still empty → clear error.
- [ ] 3.7 Fixtures: a text PDF and a table PDF; tests for both paths (table tests under `pdf` tag).
- [ ] 3.8 **DoD**: default build extracts text; `-tags pdf` build extracts tables; memory stays bounded on a large PDF.

---

## Phase 4 — OCR (`ocr` tag)

Goal: text from images and scanned PDFs. **Done (no cgo)** via a `tesseract` shell-out: images are OCR'd (`ocr.go`), scanned PDFs are rasterized with `pdftoppm` then OCR'd, and it degrades gracefully when `tesseract` is absent. **Verified end-to-end with tesseract 5.5.2** (image OCR + scanned-PDF OCR tests).

- [ ] 4.1 Verify OCR backend (🔎 `github.com/otiai10/gosseract/v2` cgo, or `tesseract` CLI shell-out); define an `OCR` interface so either backend plugs in.
- [ ] 4.2 `image_ocr.go` (`//go:build ocr`): OCR an image → text; replaces/augments the metadata-only image converter when built with `ocr`.
- [ ] 4.3 Scanned-PDF path (`pdf && ocr`): detect pages with no extractable text → rasterize via PDFium → OCR.
- [ ] 4.4 Flags: `--ocr-lang` (default `eng`), confidence threshold; clear error if built without `ocr`.
- [ ] 4.5 Fixtures (a scanned image, a scanned PDF page) + tests gated by tag.
- [ ] 4.6 **DoD**: `-tags ocr` build reads text from a scanned image; `-tags "pdf ocr"` reads a scanned PDF.

---

## Phase 5 — Media (`media` tag)

Goal: metadata + speech transcription for audio/video. **Done (no cgo)** via an `ffprobe` metadata block (format/duration/streams) in `media.go`, with an optional `whisper` transcript hook. ffmpeg/ffprobe verified here; whisper is gated/auto-detected.

- [ ] 5.1 Verify audio-tag lib (🔎 `github.com/dhowden/tag`); `audio_meta.go`: emit metadata front-matter.
- [ ] 5.2 Define `Transcriber` interface; implement a local backend (🔎 whisper.cpp binding) **and** a cloud backend, runtime-selectable via `--transcribe-engine`.
- [ ] 5.3 `audio.go`: metadata + transcript.
- [ ] 5.4 `video.go`: detect `ffmpeg`; demux audio track; transcribe; include media metadata.
- [ ] 5.5 Graceful degradation: no engine/binary available → metadata-only + a clear note.
- [ ] 5.6 Tests with a short clip (transcription test may be marked slow/optional).
- [ ] 5.7 **DoD**: `-tags media` build transcribes a short wav; video path works when `ffmpeg` present.

---

## Phase 6 — Web ingestion & optional intelligence (`cloud` tag)

Goal: convert remote documents safely; pluggable AI backends.

- [ ] 6.1 `internal/app/fetch.go`: SSRF-safe HTTP client — scheme allowlist (`http,https,file,data`), block private/loopback/link-local/metadata IPs, cap redirects, request timeout + max body size.
- [ ] 6.2 URL → response → `StreamInfo` (parse `Content-Type`, `Content-Disposition`) → dispatch.
- [ ] 6.3 `--url` CLI input and `file:`/`data:` URI handling.
- [ ] 6.4 Optional LLM image caption provider (`//go:build cloud`): env-key auth, opt-in flag, used by image converter when enabled.
- [ ] 6.5 Pluggable `DocumentAI` provider interface (`cloud`) for higher-fidelity backends; route selected file types when configured.
- [ ] 6.6 Optional `LayoutModel` interface (`cloud`) for ML layout analysis on PDFs/images.
- [ ] 6.7 Tests: SSRF guard unit tests (private IPs rejected), provider interface mocks.
- [ ] 6.8 **DoD**: `distill --url <doc-url>` converts; private-IP fetch is refused; `cloud` build exposes provider flags.

---

## Phase 7 — Product surfaces

Goal: the wedge — batch, structured output, server, agent tooling, plugins.

- [x] 7.1 `internal/app/batch.go` + `distill batch`: concurrent dir-tree walk (worker pool sized to `GOMAXPROCS`), `--out-dir` mirroring input layout, `--workers`, continue-on-error with a converted/failed summary. (include/exclude globs deferred.)
- [x] 7.2 JSON document model: `--json` serializes `convert.Result` (markdown + title today; headings/tables as richer converters land); `batch --json` writes `.json` sidecars.
- [x] 7.3 `internal/app/serve.go` + `distill serve`: `POST /convert` (raw body) + `GET /healthz`; loopback-default bind, **non-loopback refused without an auth token** (Bearer / X-Auth-Token, constant-time compare), `--max-bytes` cap, read/write/idle timeouts. (multipart upload deferred.)
- [ ] 7.4 `distill mcp`: MCP server over stdio exposing a `convert` tool (path or bytes → Markdown/JSON).
- [ ] 7.5 Plugin API: define a stable out-of-process plugin protocol (or registration hook); `--use-plugins`, `--list-plugins`.
- [ ] 7.6 Tests: batch over a temp tree, sidecar schema, serve handler (auth + limits), mcp tool call.
- [ ] 7.7 **DoD**: `distill batch ./in --out-dir ./out --json` converts a tree concurrently; `serve` rejects unauthenticated non-loopback requests.

---

## Phase 8 — Packaging & release

Goal: shippable, cross-platform, honestly documented.

- [ ] 8.1 `Makefile`/`Taskfile`: targets per tier (`core`, default `core+html+xlsx`, `full`); print binary sizes.
- [ ] 8.2 Cross-compile matrix (linux/darwin/windows × amd64/arm64); document cgo needs for `pdf`/`ocr`/`media`.
- [ ] 8.3 GoReleaser config: archives, checksums, SBOM, changelog.
- [ ] 8.4 Container: multi-stage build on distroless; publish image.
- [ ] 8.5 Homebrew tap + Scoop manifest.
- [ ] 8.6 README + docs: quick start, capability/tier matrix, **honest fidelity limits** (esp. PDF tables, OCR, transcription), security guidance for `serve`/URL fetch.
- [ ] 8.7 CI: `golangci-lint`, `go test -race`, coverage gate (≥80% core), tag-matrix build, release workflow.
- [ ] 8.8 **DoD**: tagged release produces signed cross-platform binaries + container; docs published.

---

## Cross-cutting (maintain throughout)

- [ ] C.1 Keep `testdata/` fixtures tiny and license-clean (self-authored).
- [ ] C.2 Table-driven tests; run with `-race`; deterministic output assertions for text formats.
- [ ] C.3 Security guards (SSRF, zip-slip/bomb, serve auth) covered by unit tests, not just code.
- [ ] C.4 Every new converter updates the README format matrix in the same PR.
- [ ] C.5 No dependency added without a verification note (lib, version, why).

---

### Current status
- [x] PRD authored (`PRD.md`).
- [x] Task breakdown authored (`TASKS.md`).
- [x] Phase 0 — core engine + CLI + plain-text/CSV.
- [x] Phase 1 — text family (JSON, YAML/TOML/INI, XML, RSS/Atom, ipynb, HTML).
- [x] Phase 2 — DOCX, PPTX, ODT/ODS/ODP, XLSX, image metadata, EML, ZIP/TAR, EPUB. Deferred: MSG, legacy XLS.
- [x] Phase 3A — PDF pure-Go text extraction (default build).
- [x] Phase 3B/4/5 (shell-out, no cgo) — `pdftotext` PDF, `tesseract` OCR (images + scanned PDF, **verified** with 5.5.2), `ffprobe` audio/video metadata + `whisper` hook. Auto-detected; whisper absent here. PDFium + table reconstruction (cgo) still optional/deferred.
- [~] Phase 7 — done: concurrent batch mode, JSON document model, secured `serve`. Remaining: MCP server, plugin API.
- [ ] **Next: Phase 7 cont.** — `mcp` (stdio convert tool) and the plugin API.
