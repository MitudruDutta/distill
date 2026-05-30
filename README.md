# distill

A fast, single-binary document → Markdown engine written in Go, built for LLM/RAG
pipelines. No interpreter, no heavy runtime — just one binary.

> Status: early development. See [`PRD.md`](PRD.md) for the product spec and
> [`TASKS.md`](TASKS.md) for the phased build plan.

## Build

```bash
go build -o distill ./cmd/distill
```

## Usage

```bash
distill path/to/file.csv             # Markdown to stdout
distill path/to/file.csv -o out.md   # write to a file
cat data.tsv | distill -x tsv        # read from stdin
```

Flags: `-o` output file · `-x` extension hint · `-m` MIME-type hint · `-c` charset hint.

## Supported formats

| Status | Formats |
|--------|---------|
| Available | plain text, CSV, TSV, JSON, YAML/TOML/INI, XML, RSS/Atom, Jupyter (`.ipynb`), HTML, DOCX, PPTX, XLSX, ODT/ODS/ODP, images (metadata) |
| Planned | EML, ZIP/TAR, EPUB, MSG, legacy XLS, PDF, audio/video — see [`TASKS.md`](TASKS.md) |

## Development

```bash
go vet ./...
go test ./... -race
```

## License

Not yet chosen. Add a `LICENSE` (e.g. MIT or Apache-2.0) before publishing.
