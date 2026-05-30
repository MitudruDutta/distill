package converters_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
	. "github.com/MitudruDutta/distill/internal/converters/src"
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
	if res.Markdown != "# Title\n\nBody para" {
		t.Fatalf("got %q", res.Markdown)
	}
}

func TestODFRespectsOutlineLevelForHeadings(t *testing.T) {
	content := `<?xml version="1.0"?><office:document-content xmlns:office="urn:o" xmlns:text="urn:t">` +
		`<office:body><office:text>` +
		`<text:h text:outline-level="1">Big</text:h>` +
		`<text:h text:outline-level="2">Smaller</text:h>` +
		`<text:h text:outline-level="3">Smallest</text:h>` +
		`</office:text></office:body></office:document-content>`
	data := zipBytes(t, map[string]string{"content.xml": content})
	res, _ := (ODF{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".odt"})
	for _, want := range []string{"# Big", "## Smaller", "### Smallest"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}

func TestODFEmitsListsWithNesting(t *testing.T) {
	content := `<?xml version="1.0"?><office:document-content xmlns:office="urn:o" xmlns:text="urn:t">` +
		`<office:body><office:text>` +
		`<text:list>` +
		`<text:list-item><text:p>One</text:p></text:list-item>` +
		`<text:list-item><text:list><text:list-item><text:p>Sub</text:p></text:list-item></text:list></text:list-item>` +
		`<text:list-item><text:p>Two</text:p></text:list-item>` +
		`</text:list>` +
		`</office:text></office:body></office:document-content>`
	data := zipBytes(t, map[string]string{"content.xml": content})
	res, _ := (ODF{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".odt"})
	for _, want := range []string{"- One", "  - Sub", "- Two"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}

func TestODFExtractsTable(t *testing.T) {
	content := `<?xml version="1.0"?><office:document-content xmlns:office="urn:o" xmlns:text="urn:t" xmlns:table="urn:tbl">` +
		`<office:body><office:text>` +
		`<table:table>` +
		`<table:table-row><table:table-cell><text:p>A</text:p></table:table-cell><table:table-cell><text:p>B</text:p></table:table-cell></table:table-row>` +
		`<table:table-row><table:table-cell><text:p>1</text:p></table:table-cell><table:table-cell><text:p>2</text:p></table:table-cell></table:table-row>` +
		`</table:table>` +
		`</office:text></office:body></office:document-content>`
	data := zipBytes(t, map[string]string{"content.xml": content})
	res, _ := (ODF{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".odt"})
	for _, want := range []string{"| A | B |", "| --- | --- |", "| 1 | 2 |"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}
