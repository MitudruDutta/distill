# MCP server

`distill mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io)
server over stdio (line-delimited JSON-RPC 2.0). Any MCP-compatible client
(Claude Desktop, Kiro CLI, custom agents, …) can use distill as a
**document-conversion tool**.

## Tool: `convert`

The server exposes a single tool:

```jsonc
{
  "name": "convert",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path":   { "type": "string",
                  "description": "Absolute path to the file to convert." },
      "format": { "type": "string", "enum": ["markdown", "json"],
                  "description": "Output format (default: markdown)." }
    },
    "required": ["path"]
  }
}
```

Returns the converted document as `text` content. On errors (missing path,
unsupported format, conversion failure) returns `isError: true` with the
error message — the agent sees a tool failure to react to, not a JSON-RPC
protocol error.

## Supported MCP methods

| Method | Behavior |
|--------|----------|
| `initialize` | Returns `protocolVersion: 2024-11-05`, advertises `tools` capability |
| `notifications/initialized` | Acknowledged silently |
| `tools/list` | Returns the `convert` tool descriptor |
| `tools/call` | Executes `convert` |
| `ping` | Empty result |

Unknown methods return JSON-RPC error code `-32601`. Malformed JSON returns
`-32700`. Notifications (no `id`) never get a response.

## Setup — Kiro CLI

```jsonc
// ~/.kiro/settings/mcp.json   (or .kiro/settings/mcp.json for workspace scope)
{
  "mcpServers": {
    "distill": {
      "command": "/home/USER/.local/bin/distill",
      "args": ["mcp"]
    }
  }
}
```

Then in any new chat session, the `convert` tool appears (typically prefixed
as `@distill/convert`). Use `/mcp` to inspect status.

## Setup — Claude Desktop

```jsonc
// ~/.config/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "distill": {
      "command": "/usr/local/bin/distill",
      "args": ["mcp"]
    }
  }
}
```

## Setup — generic MCP client

Any client that supports MCP stdio servers can wire distill in by spawning
`distill mcp` and speaking JSON-RPC 2.0. The server is tested with the
`2024-11-05` protocol version.

## Performance notes

- The MCP server is a **single long-lived process** per client session. The
  PDFium WASM engine (when built with `-tags pdfium`) initializes on the first
  PDF call (~1 s) and stays warm for subsequent calls.
- Converters share the same registry the CLI uses, so anything `distill FILE`
  converts works identically through MCP.
- Large files: the binary reads the file fully into memory, so size scales
  with available RAM. For multi-GB inputs, use the CLI/server modes with
  streaming-friendly tooling instead.

## Security

The MCP server reads files from your local filesystem with **the privileges
of the process that spawned it** (your user). It does not validate paths
against any allowlist — your MCP client is the trust boundary. If you don't
trust a client, don't wire distill into it.
