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

- [ ] 2.1 `internal/converters/ooxml/zip.go`: helper to open a zip from an `io.Reader` (buffer if non-seekable) and read a named entry.
- [ ] 2.2 `docx.go`: parse `word/document.xml` — paragraphs (`w:p`), runs (`w:r/w:t`), heading styles → `#`, tables (`w:tbl`), lists (numbering) → `-`/`1.`.
- [ ] 2.3 `pptx.go`: iterate `ppt/slides/slideN.xml` in order; slide title → `##`; body text (`a:t`) → paragraphs/bullets; optional `--slide-separators`.
- [ ] 2.4 Add `github.com/xuri/excelize/v2` ✅; `xlsx.go`: each sheet → `## <name>` + Markdown table; handle `.xlsx` and `.xls`.
- [ ] 2.5 `odf.go`: ODT/ODS/ODP via `content.xml` — `text:h`→headings, `text:p`→paragraphs, `table:table`→tables.
- [ ] 2.6 `eml.go`: `net/mail` headers (From/To/Subject/Date) as front-matter; walk `multipart`; HTML parts via the Phase-1 HTML pipeline; prefer text/plain else text/html.
- [ ] 2.7 Verify + add OLE/CFB reader for MSG (🔎 e.g. `github.com/richardlehane/mscfb`); `msg.go`: extract subject/sender/body.
- [ ] 2.8 `archive.go`: ZIP/TAR — iterate entries, recurse through the engine; enforce zip-slip guard, per-entry + total size caps, max entry count, recursion-depth cap; emit a heading per entry.
- [ ] 2.9 `epub.go`: read OPF spine order; convert each XHTML doc via HTML pipeline; concatenate with chapter headings.
- [ ] 2.10 Verify + add EXIF (`github.com/rwcarlsen/goexif` 🔎); `image_meta.go`: dimensions + EXIF table (no pixels yet).
- [ ] 2.11 Fixtures + tests for every format above (use tiny real files).
- [ ] 2.12 **DoD**: tags `core,html,xlsx` compile; all tests pass; a real `.docx/.xlsx/.pptx/.epub` converts.

---

## Phase 3 — PDF (text → PDFium → tables)

Goal: solid text extraction by default, high-fidelity + tables behind the `pdf` (cgo) tag.

- [ ] 3.1 `pdf_purego.go` (no `pdf` tag): pure-Go text via `github.com/ledongthuc/pdf` ✅ or `rsc.io/pdf` ✅; whole-document text extraction.
- [ ] 3.2 Verify + add `github.com/klippa-app/go-pdfium` ✅; `pdf_pdfium.go` (`//go:build pdf`): init instance pool, open from bytes, extract text per page, close pages to bound memory.
- [ ] 3.3 Build-tag dispatch: `pdf` tag overrides pure-Go converter at registration.
- [ ] 3.4 Word-box extraction from PDFium (position + size per word) for layout work.
- [ ] 3.5 Table reconstruction: cluster words into rows (Y) and columns (X) with adaptive tolerance; render aligned Markdown tables; pass through non-table text as paragraphs.
- [ ] 3.6 Fallback chain: PDFium fails/empty → pure-Go path; still empty → clear error.
- [ ] 3.7 Fixtures: a text PDF and a table PDF; tests for both paths (table tests under `pdf` tag).
- [ ] 3.8 **DoD**: default build extracts text; `-tags pdf` build extracts tables; memory stays bounded on a large PDF.

---

## Phase 4 — OCR (`ocr` tag)

Goal: text from images and scanned PDFs.

- [ ] 4.1 Verify OCR backend (🔎 `github.com/otiai10/gosseract/v2` cgo, or `tesseract` CLI shell-out); define an `OCR` interface so either backend plugs in.
- [ ] 4.2 `image_ocr.go` (`//go:build ocr`): OCR an image → text; replaces/augments the metadata-only image converter when built with `ocr`.
- [ ] 4.3 Scanned-PDF path (`pdf && ocr`): detect pages with no extractable text → rasterize via PDFium → OCR.
- [ ] 4.4 Flags: `--ocr-lang` (default `eng`), confidence threshold; clear error if built without `ocr`.
- [ ] 4.5 Fixtures (a scanned image, a scanned PDF page) + tests gated by tag.
- [ ] 4.6 **DoD**: `-tags ocr` build reads text from a scanned image; `-tags "pdf ocr"` reads a scanned PDF.

---

## Phase 5 — Media (`media` tag)

Goal: metadata + speech transcription for audio/video.

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

- [ ] 7.1 `internal/app/batch.go` + `distill batch`: walk a dir tree, worker pool sized to `GOMAXPROCS`, `--out-dir` mirroring input layout, include/exclude globs, progress + aggregated error report, continue-on-error.
- [ ] 7.2 JSON sidecar: serialize `Result` (title, headings tree, tables, detected type, source metadata); `--json` (stdout) and batch `.json` next to each `.md`.
- [ ] 7.3 `internal/app/serve.go` + `distill serve`: `POST /convert` (multipart upload + raw body), `127.0.0.1` default bind, **required auth token** for non-loopback, body-size limit, per-request timeout, `/healthz`.
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
- [ ] **Next: Phase 2** — Office/ODF/e-mail/archives/e-books/image metadata.
