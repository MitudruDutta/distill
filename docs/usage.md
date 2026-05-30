# Usage

distill has four modes, all exposed by the same single binary:

| Mode | Command | What it does |
|------|---------|--------------|
| One-shot | `distill FILE` | Convert one file to Markdown on stdout |
| Batch | `distill batch DIR --out-dir OUT` | Concurrent directory conversion |
| Server | `distill serve --addr 127.0.0.1:8080` | HTTP `POST /convert` endpoint |
| MCP | `distill mcp` | stdio Model Context Protocol server |

## One-shot conversion

```bash
distill report.docx                    # → Markdown on stdout
distill report.docx -o report.md       # → write to a file
cat data.csv | distill -x csv          # → stdin, with extension hint
distill report.docx -json              # → JSON document model
distill report.docx -json -o out.json  # → JSON to file
```

### Flags

| Flag | Description |
|------|-------------|
| `-o PATH` | Output file (default: stdout) |
| `-x EXT` | Extension hint, useful with stdin (`csv`, `tsv`, `json`, `pdf`, …) |
| `-m MIME` | MIME-type hint |
| `-c CHARSET` | Charset hint (e.g. `utf-8`) |
| `-json` | Emit a JSON document model instead of Markdown |

The flag parser accepts flags **before or after** the filename:
`distill file.tsv -o out.md` and `distill -o out.md file.tsv` both work.

## Batch mode

```bash
distill batch ./docs --out-dir ./md           # concurrent dir tree → mirrored .md files
distill batch ./docs --out-dir ./json --json  # write .json sidecars instead
distill batch ./docs --out-dir ./md --workers 8
```

- Walks the input directory, converting every regular file in parallel.
- Worker pool sized to `GOMAXPROCS` by default; override with `--workers`.
- Output mirrors the input layout (`docs/sub/foo.pdf` → `md/sub/foo.md`).
- **Continue-on-error**: prints a summary like `converted 47, failed 3`.

Use this whenever you have a corpus to ingest into a RAG pipeline.

## Server mode

```bash
distill serve --addr 127.0.0.1:8080
distill serve --addr 0.0.0.0:9090 --token "$DISTILL_TOKEN" --max-bytes 64000000
```

Endpoints:

- `POST /convert` — body is the raw file bytes; returns Markdown.
  Optional query params: `?ext=docx` (extension hint), `?format=json`
  (return JSON model). Set the `Content-Type` header for the MIME hint.
- `GET /healthz` — liveness check.

Auth: `Authorization: Bearer <token>` or `X-Auth-Token: <token>`.

```bash
curl --data-binary @report.docx \
  -H "Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document" \
  http://127.0.0.1:8080/convert?format=json
```

**Security defaults** (see [security.md](security.md)):

- Binds **`127.0.0.1`** by default.
- **Refuses to bind a non-loopback address without `--token`** (or
  `DISTILL_TOKEN` env var). No accidental open service.
- Body capped via `--max-bytes` (32 MiB default).
- Read/write/idle timeouts on every request.

## MCP mode

```bash
distill mcp   # speaks line-delimited JSON-RPC 2.0 on stdin/stdout
```

For agent integration. See [mcp.md](mcp.md) and [agents.md](agents.md).

## JSON document model (`-json`)

Every mode that emits Markdown can also emit a structured JSON model:

```json
{
  "markdown": "# Title\n\n- bullet\n",
  "title": "Title",
  "headings": ["Title"],
  "tables": [{"header": ["a","b"], "rows": [["1","2"]]}]
}
```

Use this when feeding a downstream chunker that needs structure (for now
`headings` and `tables` are populated by a subset of converters; `markdown` is
always present).

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Conversion or I/O error (message printed to stderr) |
