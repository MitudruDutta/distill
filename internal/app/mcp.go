package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/MitudruDutta/distill/internal/convert"
)

// MCP protocol version we advertise. Echoed back during initialize so clients
// see we accept their version negotiation; agents that pin a different version
// either accept ours or fail fast.
const mcpProtocolVersion = "2024-11-05"

// distillVersion is reported in the MCP serverInfo. Bump on real releases.
const distillVersion = "0.1.0"

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP runs a Model Context Protocol server over stdin/stdout (line-delimited
// JSON-RPC 2.0). It exposes a single "convert" tool that converts any
// supported file at a local path to Markdown via reg.
//
// Returns nil on EOF (clean shutdown) or a scanner error.
func MCP(reg *convert.Registry, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// MCP messages are usually small, but allow up to 16 MiB per line in case a
	// client embeds a large payload.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var req mcpRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(mcpResponse{
				JSONRPC: "2.0",
				Error:   &mcpError{Code: -32700, Message: "parse error: " + err.Error()},
			})
			continue
		}
		// JSON-RPC notifications have no id and expect no response.
		isNotification := len(req.ID) == 0 || string(req.ID) == "null"

		switch req.Method {
		case "initialize":
			_ = enc.Encode(mcpResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]any{
					"protocolVersion": mcpProtocolVersion,
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo": map[string]any{
						"name":    "distill",
						"version": distillVersion,
					},
				},
			})

		case "notifications/initialized", "initialized":
			// Acknowledged via no-op (notification, no response expected).

		case "tools/list":
			_ = enc.Encode(mcpResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]any{
					"tools": []any{convertToolDescriptor()},
				},
			})

		case "tools/call":
			result, err := mcpHandleToolCall(reg, req.Params)
			if err != nil {
				// Tool errors are reported via isError:true (not a JSON-RPC
				// error), so the agent sees the failure as tool output.
				_ = enc.Encode(mcpResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]any{
						"content": []any{map[string]any{"type": "text", "text": err.Error()}},
						"isError": true,
					},
				})
			} else {
				_ = enc.Encode(mcpResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  result,
				})
			}

		case "ping":
			_ = enc.Encode(mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})

		default:
			if !isNotification {
				_ = enc.Encode(mcpResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &mcpError{Code: -32601, Message: "method not found: " + req.Method},
				})
			}
		}
	}
	return scanner.Err()
}

// convertToolDescriptor returns the MCP tools/list entry for the convert tool.
func convertToolDescriptor() map[string]any {
	return map[string]any{
		"name":        "convert",
		"description": "Convert a document at a local file path to Markdown. Supports text, CSV/TSV, JSON, YAML/TOML/INI, XML, RSS/Atom, Jupyter, HTML, DOCX, PPTX, XLSX, ODF, EML, ZIP/TAR, EPUB, PDF, images, and audio/video.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Absolute path to the file to convert.",
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"markdown", "json"},
					"description": "Output format. 'markdown' (default) returns the document text; 'json' returns the structured Result with title, headings, and tables fields where available.",
				},
			},
			"required": []string{"path"},
		},
	}
}

// mcpHandleToolCall executes a tools/call invocation. Returns the result
// payload (with content array) on success, or an error to be surfaced as
// isError:true content on failure.
func mcpHandleToolCall(reg *convert.Registry, params json.RawMessage) (any, error) {
	var p struct {
		Name      string `json:"name"`
		Arguments struct {
			Path   string `json:"path"`
			Format string `json:"format"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Name != "convert" {
		return nil, fmt.Errorf("unknown tool: %s", p.Name)
	}
	if p.Arguments.Path == "" {
		return nil, errors.New("missing required argument: path")
	}

	data, err := os.ReadFile(p.Arguments.Path)
	if err != nil {
		return nil, err
	}
	peek := data
	if len(peek) > 512 {
		peek = peek[:512]
	}
	base := convert.StreamInfo{
		Filename:  p.Arguments.Path,
		LocalPath: p.Arguments.Path,
		Extension: convert.ExtensionOf(p.Arguments.Path),
	}
	res, err := reg.Convert(bytes.NewReader(data), convert.Guess(base, peek))
	if err != nil {
		return nil, err
	}

	var text string
	if p.Arguments.Format == "json" {
		b, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return nil, err
		}
		text = string(b)
	} else {
		text = res.Markdown
	}
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
	}, nil
}
