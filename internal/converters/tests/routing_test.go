package converters_test

import (
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestDispatchHTMLBeatsPlainText(t *testing.T) {
	res, err := Default().Convert(strings.NewReader("<h1>Hi</h1>"),
		[]convert.StreamInfo{{Extension: ".html", Mimetype: "text/html"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# Hi") {
		t.Fatalf("expected HTML conversion, got %q", res.Markdown)
	}
}

func TestDispatchNonFeedXMLFallsThroughToFence(t *testing.T) {
	res, err := Default().Convert(strings.NewReader("<note><to>x</to></note>"),
		[]convert.StreamInfo{{Extension: ".xml", Mimetype: "text/xml"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Markdown, "```xml") {
		t.Fatalf("expected xml fence, got %q", res.Markdown)
	}
}
