package converters_test

import (
	"bytes"
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestEPUBConvertsSpineChapters(t *testing.T) {
	container := `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` +
		`<rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`
	opf := `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><manifest>` +
		`<item id="c1" href="chap1.xhtml" media-type="application/xhtml+xml"/>` +
		`</manifest><spine><itemref idref="c1"/></spine></package>`
	chap := `<html><body><h1>Chapter 1</h1><p>Once upon a time.</p></body></html>`
	data := zipBytes(t, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      opf,
		"OEBPS/chap1.xhtml":      chap,
	})
	res, err := (EPUB{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".epub"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Chapter 1", "Once upon a time."} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}
