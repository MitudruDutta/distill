# Security

distill processes potentially-untrusted input (uploaded files, archives,
documents fetched over HTTP). This page lists what's hardened and where the
trust boundaries are.

## Trust model

- The **agent / caller** is the trust boundary. distill itself does not
  authenticate users; if it's wired into a tool, that tool is trusted.
- The **input file** is untrusted. distill must parse safely.
- File I/O runs with **process privileges**. If you run distill as your user,
  it can read everything you can — same as `cat`. There is no allowlist.

## What's hardened

### Server mode (`distill serve`)

- Loopback bind by default.
- **Refuses non-loopback bind without an auth token** (`--token` or
  `DISTILL_TOKEN`). No accidental open service.
- Bearer token auth with constant-time compare.
- `MaxBytesReader` body cap (configurable via `--max-bytes`).
- Read/write/idle timeouts on every connection.

### Archive converters (ZIP / TAR / EPUB)

- **Per-entry byte cap** (16 MiB).
- **Total bytes cap** across all entries (128 MiB).
- **Entry count cap** (2,000 entries).
- **No disk writes** during entry recursion (zip-slip not applicable).
- **Nested archives are skipped**, bounding recursion depth.

### PDF / OCR / media pipelines

- Pure-Go PDF reader is **panic-recovered**: malformed input returns an
  error instead of crashing the process.
- External tool invocations (`pdftotext`, `tesseract`, `ffprobe`, `whisper`)
  use `exec.CommandContext` with a 120-second timeout.
- Temp files used by OCR/whisper pipelines are written to `os.TempDir()` and
  removed on exit.

### MCP server

- Reads files from the local filesystem **only when** the agent calls
  `convert` with a path. No magic auto-loading.
- Returns tool errors as `isError: true` content, never as JSON-RPC errors,
  so the agent sees the failure as tool output.

## What's NOT done (be honest)

- **No path allowlisting.** A misconfigured agent could ask distill to read
  any file your user has access to. Limit the agent's autonomy if this matters.
- **No sandboxing of external tools.** `tesseract`, `whisper`, `ffmpeg`,
  `pdftotext` run as your user. Keep them updated.
- **URL fetching is implemented and SSRF-guarded** (see "URL fetching" below).
  Loopback, private (RFC 1918 / 4193), link-local (incl. cloud-metadata
  169.254.169.254), multicast, and unspecified IPs are refused before
  connect. The check runs after DNS resolution on every redirect hop, so
  DNS rebinding cannot bypass it. Schemes are allowlisted (`http`, `https`,
  `file`, `data`); body size, redirect count, and request timeout are capped.

## Reporting a vulnerability

Open a GitHub issue tagged `security`, or email the maintainer for sensitive
disclosures.
