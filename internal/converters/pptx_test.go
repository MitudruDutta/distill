package converters

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestPPTXExtractsSlidesInOrder(t *testing.T) {
	slide := func(text string) string {
		return `<?xml version="1.0"?><p:sld xmlns:a="urn:a" xmlns:p="urn:p"><p:cSld><p:spTree>` +
			`<a:p><a:r><a:t>` + text + `</a:t></a:r></a:p>` +
			`</p:spTree></p:cSld></p:sld>`
	}
	data := zipBytes(t, map[string]string{
		"ppt/slides/slide1.xml": slide("First slide"),
		"ppt/slides/slide2.xml": slide("Second slide"),
	})
	res, err := (PPTX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".pptx"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Slide 1", "First slide", "## Slide 2", "Second slide"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
	if strings.Index(res.Markdown, "First") > strings.Index(res.Markdown, "Second") {
		t.Fatal("slides emitted out of order")
	}
}
