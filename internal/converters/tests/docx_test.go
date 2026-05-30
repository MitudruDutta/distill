package converters_test

import (
	"archive/zip"
	"bytes"
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
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
