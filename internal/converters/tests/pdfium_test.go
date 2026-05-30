//go:build pdfium

package converters_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-pdf/fpdf"

	"github.com/MitudruDutta/distill/internal/convert"
	. "github.com/MitudruDutta/distill/internal/converters/src"
)

func TestPDFiumExtractsText(t *testing.T) {
	gen := fpdf.New("P", "pt", "Letter", "")
	gen.SetCompression(false)
	gen.AddPage()
	gen.SetFont("Helvetica", "", 24)
	gen.Cell(200, 30, "Hello PDFium")
	var buf bytes.Buffer
	if err := gen.Output(&buf); err != nil {
		t.Fatal(err)
	}
	res, err := (PDFium{}).Convert(&buf, convert.StreamInfo{Extension: ".pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ReplaceAll(res.Markdown, " ", ""), "HelloPDFium") {
		t.Fatalf("got %q", res.Markdown)
	}
}
