package converters

import (
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestIpynbRendersCellsAndOutputs(t *testing.T) {
	in := `{"cells":[
	  {"cell_type":"markdown","source":["# Title\n","para"]},
	  {"cell_type":"code","source":"print(1)","outputs":[{"output_type":"stream","text":["out\n"]}]},
	  {"cell_type":"code","source":["x=1"],"outputs":[{"output_type":"execute_result","data":{"text/plain":["42"]}}]},
	  {"cell_type":"raw","source":["should be ignored"]}
	],"metadata":{"language_info":{"name":"go"}}}`
	res, err := (Ipynb{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".ipynb"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Title", "```go\nprint(1)\n```", "out", "```go\nx=1\n```", "42"} {
		if !strings.Contains(res.Markdown, want) {
			t.Errorf("missing %q in:\n%s", want, res.Markdown)
		}
	}
	if strings.Contains(res.Markdown, "should be ignored") {
		t.Error("raw cell content should not appear")
	}
}

func TestIpynbDefaultsToPythonWhenLanguageMissing(t *testing.T) {
	res, _ := (Ipynb{}).Convert(strings.NewReader(`{"cells":[{"cell_type":"code","source":"x"}]}`), convert.StreamInfo{Extension: ".ipynb"})
	if !strings.Contains(res.Markdown, "```python\nx\n```") {
		t.Fatalf("expected python fallback, got:\n%s", res.Markdown)
	}
}
