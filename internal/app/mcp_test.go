package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	converters "github.com/MitudruDutta/distill/internal/converters/src"
)

// readMCPResponses splits encoder output into one decoded message per non-empty
// line so tests can assert on each independently.
func readMCPResponses(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, ln := range bytes.Split(raw, []byte("\n")) {
		if len(bytes.TrimSpace(ln)) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(ln, &m); err != nil {
			t.Fatalf("unparseable response line %q: %v", string(ln), err)
		}
		out = append(out, m)
	}
	return out
}

func runMCPRequests(t *testing.T, requests ...string) []map[string]any {
	t.Helper()
	var in bytes.Buffer
	for _, r := range requests {
		in.WriteString(r + "\n")
	}
	var out bytes.Buffer
	if err := MCP(converters.Default(), &in, &out); err != nil {
		t.Fatal(err)
	}
	return readMCPResponses(t, out.Bytes())
}

func TestMCPInitializeReturnsProtocolAndCapabilities(t *testing.T) {
	resps := runMCPRequests(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if len(resps) != 1 {
		t.Fatalf("want 1 response, got %d", len(resps))
	}
	result, ok := resps[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", resps[0])
	}
	if pv, _ := result["protocolVersion"].(string); pv == "" {
		t.Errorf("missing protocolVersion in: %v", result)
	}
	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("no capabilities: %v", result)
	}
	if _, ok := caps["tools"]; !ok {
		t.Errorf("must advertise tools capability: %v", caps)
	}
	srv, _ := result["serverInfo"].(map[string]any)
	if name, _ := srv["name"].(string); name != "distill" {
		t.Errorf("serverInfo.name = %q, want distill", name)
	}
}

func TestMCPToolsListExposesConvertWithSchema(t *testing.T) {
	resps := runMCPRequests(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := resps[0]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "convert" {
		t.Errorf("tool.name = %v, want convert", tool["name"])
	}
	schema := tool["inputSchema"].(map[string]any)
	required := schema["required"].([]any)
	if len(required) != 1 || required[0] != "path" {
		t.Errorf("required = %v, want [path]", required)
	}
	props := schema["properties"].(map[string]any)
	if _, ok := props["path"]; !ok {
		t.Errorf("schema missing 'path' property: %v", props)
	}
}

func TestMCPToolsCallConvertsCSVToMarkdown(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "t.csv")
	if err := os.WriteFile(csv, []byte("name,age\nAda,36\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"name":      "convert",
		"arguments": map[string]any{"path": csv},
	})
	req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":` + string(body) + `}`
	resps := runMCPRequests(t, req)
	result := resps[0]["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("unexpected error result: %v", result)
	}
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "| name | age |") {
		t.Fatalf("expected Markdown table in tool result, got:\n%s", text)
	}
}

func TestMCPToolsCallMissingPathReturnsIsError(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"name": "convert", "arguments": map[string]any{}})
	req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":` + string(body) + `}`
	resps := runMCPRequests(t, req)
	result := resps[0]["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("want isError true on missing path, got: %v", result)
	}
}

func TestMCPUnknownMethodReturnsJSONRPCError(t *testing.T) {
	resps := runMCPRequests(t, `{"jsonrpc":"2.0","id":5,"method":"does/not/exist"}`)
	errObj, ok := resps[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected JSON-RPC error, got: %v", resps[0])
	}
	if code, _ := errObj["code"].(float64); int(code) != -32601 {
		t.Errorf("code = %v, want -32601 (method not found)", code)
	}
}

func TestMCPNotificationReceivesNoResponse(t *testing.T) {
	// initialized notification (no id) must be silently consumed.
	resps := runMCPRequests(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(resps) != 2 {
		t.Fatalf("want 2 responses (initialize + tools/list), got %d", len(resps))
	}
}

func TestMCPMalformedJSONGetsParseError(t *testing.T) {
	resps := runMCPRequests(t, `not even close to json`)
	errObj, ok := resps[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error response, got: %v", resps[0])
	}
	if code, _ := errObj["code"].(float64); int(code) != -32700 {
		t.Errorf("code = %v, want -32700 (parse error)", code)
	}
}
