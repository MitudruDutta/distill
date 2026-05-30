package converters_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
	. "github.com/MitudruDutta/distill/internal/converters/src"
)

// zipBytes builds an in-memory zip from name->content entries (shared by the
// DOCX/PPTX/ODF tests).
func zipBytes(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDOCXExtractsParagraphText(t *testing.T) {
	doc := `<?xml version="1.0"?><w:document xmlns:w="urn:w"><w:body>` +
		`<w:p><w:r><w:t>Hello </w:t></w:r><w:r><w:t>world</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Second</w:t></w:r></w:p>` +
		`</w:body></w:document>`
	data := zipBytes(t, map[string]string{"word/document.xml": doc})
	res, err := (DOCX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".docx"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Markdown != "Hello world\n\nSecond" {
		t.Fatalf("got %q", res.Markdown)
	}
}

func TestDOCXMissingDocumentXMLErrors(t *testing.T) {
	data := zipBytes(t, map[string]string{"other.xml": "<x/>"})
	if _, err := (DOCX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".docx"}); err == nil {
		t.Fatal("expected error when word/document.xml is absent")
	}
}

func TestDOCXEmitsHeadingsFromStyles(t *testing.T) {
	doc := `<?xml version="1.0"?><w:document xmlns:w="urn:w"><w:body>` +
		`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Big</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>Smaller</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Body text</w:t></w:r></w:p>` +
		`</w:body></w:document>`
	data := zipBytes(t, map[string]string{"word/document.xml": doc})
	res, _ := (DOCX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".docx"})
	want := "# Big\n\n## Smaller\n\nBody text"
	if res.Markdown != want {
		t.Fatalf("got %q\nwant %q", res.Markdown, want)
	}
}

func TestDOCXEmitsListsWithIndent(t *testing.T) {
	doc := `<?xml version="1.0"?><w:document xmlns:w="urn:w"><w:body>` +
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Item 1</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Sub item</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Item 2</w:t></w:r></w:p>` +
		`</w:body></w:document>`
	data := zipBytes(t, map[string]string{"word/document.xml": doc})
	res, _ := (DOCX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".docx"})
	for _, want := range []string{"- Item 1", "  - Sub item", "- Item 2"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}

func TestDOCXExtractsTable(t *testing.T) {
	doc := `<?xml version="1.0"?><w:document xmlns:w="urn:w"><w:body>` +
		`<w:tbl>` +
		`<w:tr><w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>Age</w:t></w:r></w:p></w:tc></w:tr>` +
		`<w:tr><w:tc><w:p><w:r><w:t>Ada</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>36</w:t></w:r></w:p></w:tc></w:tr>` +
		`</w:tbl>` +
		`</w:body></w:document>`
	data := zipBytes(t, map[string]string{"word/document.xml": doc})
	res, _ := (DOCX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".docx"})
	for _, want := range []string{"| Name | Age |", "| --- | --- |", "| Ada | 36 |"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}
