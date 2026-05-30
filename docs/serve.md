# HTTP server

`distill serve` exposes a small HTTP API that converts uploaded bytes to
Markdown. Useful for backend services, microservices, or tying distill into a
non-Go RAG pipeline.

## Run

```bash
distill serve --addr 127.0.0.1:8080                 # default: loopback only
distill serve --addr 0.0.0.0:9090 --token $TOKEN    # exposed; auth required
distill serve --addr :8080 --max-bytes 64000000     # 64 MiB body cap
```

`--token` may also be supplied via the `DISTILL_TOKEN` environment variable.

## Endpoints

### `POST /convert`

Body: raw file bytes.

Headers (optional):

- `Content-Type: <mime>` — type hint
- `Authorization: Bearer <token>` or `X-Auth-Token: <token>` — when configured

Query params (optional):

- `?ext=docx` — extension hint
- `?format=json` — return the JSON document model instead of Markdown

Response: Markdown (`text/markdown`) or JSON (`application/json`) depending on
`format`.

Status codes:

- `200` — success
- `401` — missing/invalid token (when configured)
- `405` — non-POST method
- `413` — body exceeded `--max-bytes`
- `422` — conversion failed (body explains why)

### `GET /healthz`

Liveness check. Returns `200 OK` with body `ok`.

## Examples

```bash
# Markdown
curl --data-binary @report.docx \
  -H "Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document" \
  http://127.0.0.1:8080/convert

# JSON
curl --data-binary @data.csv "http://127.0.0.1:8080/convert?ext=csv&format=json"

# With auth
curl --data-binary @file.pdf \
  -H "Authorization: Bearer $DISTILL_TOKEN" \
  https://distill.internal/convert
```

## Security

`serve` is **secure-by-default**:

- Binds `127.0.0.1` if `--addr` doesn't specify a host.
- **Refuses to bind a non-loopback address without `--token`** — no accidental
  open service on the network. Use `DISTILL_TOKEN` env var to keep the token
  out of process tables.
- Token comparison is constant-time.
- Per-request body cap (`MaxBytesReader`).
- Read/write/idle timeouts on every connection.

For Internet-facing deployment, terminate TLS upstream (reverse proxy /
load balancer) and forward to distill on loopback.

## Operational notes

- The process is single-binary, no model files, no scratch directories
  beyond temp files for OCR/transcription pipelines (auto-cleaned).
- Memory scales with the largest concurrent body. With `--max-bytes 32M` and N
  workers, peak memory is roughly `N × 32 MB + base`.
- For PDFium-tagged builds, the WASM engine is lazy-initialized on the first
  PDF request and stays warm for the process lifetime.
