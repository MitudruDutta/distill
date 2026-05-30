package converters_test

import (
	"bytes"
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestODFExtractsHeadingsAndParagraphs(t *testing.T) {
	content := `<?xml version="1.0"?><office:document-content xmlns:office="urn:o" xmlns:text="urn:t">` +
		`<office:body><office:text>` +
		`<text:h>Title</text:h><text:p>Body para</text:p>` +
		`</office:text></office:body></office:document-content>`
	data := zipBytes(t, map[string]string{"content.xml": content})
	res, err := (ODF{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".odt"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Markdown != "Title\n\nBody para" {
		t.Fatalf("got %q", res.Markdown)
	}
}
