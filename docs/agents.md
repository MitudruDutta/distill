# Use distill from an LLM agent

When an agent processes user-provided files (PDFs, DOCX, images, audio, …)
the highest-leverage move is to **convert to Markdown first, then reason
over the Markdown**. Markdown is what LLMs are trained on; raw bytes of
binary formats are not.

This page shows three patterns to wire that up.

## Pattern 1 — system-prompt instruction (simplest, works everywhere)

Add a clause to your agent's system prompt telling it to call
`@distill/convert` for non-text inputs **before** doing anything else.

```text
TOOL USE — DOCUMENT INPUTS

Whenever the user references a path that points to a non-plaintext document
(PDF, DOCX, PPTX, XLSX, ODT/ODS/ODP, EML, EPUB, ZIP/TAR, image, audio, or
video), CALL THE @distill/convert TOOL FIRST with that path, then reason
about the resulting Markdown. Never try to read those formats with a generic
file-reader; you will get binary garbage.

For plain-text formats (.txt, .md, .csv, .json, .yaml, .html) you can read
them directly OR call @distill/convert — both produce useful Markdown.
```

LLMs trained on tool-use follow this reliably. It's the lowest-friction
option and works for Claude Desktop, Kiro CLI, OpenAI agents — any client
where you can edit the system prompt.

## Pattern 2 — dedicated agent config (Kiro CLI)

Ship the pattern as a reusable agent. Create
`~/.kiro/agents/distill-aware.json`:

```jsonc
{
  "name": "distill-aware",
  "description": "An agent that auto-converts every non-text document via distill before reasoning.",
  "prompt": "You are an analyst. Whenever the user mentions a path that points to a non-plaintext document (PDF/DOCX/PPTX/XLSX/ODF/EML/EPUB/image/audio/video), CALL @distill/convert with that path FIRST, then reason about the resulting Markdown. Never try to read those formats directly. For tabular data, treat the resulting Markdown table as the source of truth. For audio/video, the metadata block + transcript are your inputs.",
  "tools": ["read", "grep", "glob", "shell"],
  "allowedTools": ["read", "grep", "glob", "@distill/convert"],
  "mcpServers": {
    "distill": {
      "command": "/home/USER/.local/bin/distill",
      "args": ["mcp"]
    }
  }
}
```

A copy is shipped at [`examples/agents/distill-aware.json`](../examples/agents/distill-aware.json).

Switch into it inside Kiro CLI with `/agent distill-aware`.

## Pattern 3 — preprocessing hook (deterministic)

If you want conversion to be **forced** rather than depending on the LLM
deciding to call the tool, run a `userPromptSubmit` hook that pre-converts
any file path the user mentions and prepends the Markdown to the prompt.

```jsonc
{
  "hooks": {
    "userPromptSubmit": [
      { "command": "scripts/auto-distill.sh" }
    ]
  }
}
```

Sketch of `scripts/auto-distill.sh`:

```bash
#!/usr/bin/env bash
# Read the user prompt from stdin, scan for file paths, convert each
# non-text path via distill, and emit the Markdown back as added context.
set -euo pipefail
prompt=$(cat)
for path in $(grep -oE '(/[A-Za-z0-9._/-]+|~/[A-Za-z0-9._/-]+)' <<<"$prompt"); do
  expanded="${path/#~/$HOME}"
  case "${expanded,,}" in
    *.pdf|*.docx|*.pptx|*.xlsx|*.odt|*.ods|*.odp|*.eml|*.epub|*.zip|*.tar|\
    *.png|*.jpg|*.jpeg|*.gif|*.mp3|*.wav|*.m4a|*.mp4|*.mov|*.mkv)
      if [ -f "$expanded" ]; then
        echo "<file path=\"$expanded\">"
        distill "$expanded"
        echo "</file>"
      fi
      ;;
  esac
done
echo "$prompt"
```

This guarantees the conversion happens regardless of LLM behavior.

## Which pattern to pick

| Goal | Pattern |
|------|---------|
| Quick setup, trust the LLM | 1 (system-prompt) |
| Reusable across many sessions | 2 (agent config) |
| Must-not-fail, deterministic | 3 (hook) |

Pattern 1 covers ~95% of cases. Pattern 3 is for when an LLM mistake would be
costly (compliance, financial extraction, etc.).

## Why this is worth it

Without conversion, an LLM trying to "read" a PDF or DOCX gets binary noise
or a fragmented stream of glyphs. Token costs balloon and quality drops.
With distill in front, the model sees:

```markdown
## Slide 2: Fruit Data

| Fruit   | Quantity | Price per Unit |
| ---     | ---      | ---            |
| Apples  | 5        | $0.50          |
| Mangoes | 3        | $1.00          |
```

— exactly what it was trained on. Fewer tokens, better structure, accurate
RAG retrieval.
