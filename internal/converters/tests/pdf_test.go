package converters_test

import (
	"bytes"
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/go-pdf/fpdf"
)

func TestPDFExtractsText(t *testing.T) {
	gen := fpdf.New("P", "pt", "Letter", "")
	gen.SetCompression(false)
	gen.AddPage()
	gen.SetFont("Helvetica", "", 24)
	gen.Cell(200, 30, "Hello PDF")
	var buf bytes.Buffer
	if err := gen.Output(&buf); err != nil {
		t.Fatal(err)
	}

	res, err := (PDF{}).Convert(&buf, convert.StreamInfo{Extension: ".pdf"})
	if err != nil {
		t.Fatal(err)
	}
	// Spacing between words varies by extractor; assert the content is present.
	if !strings.Contains(strings.ReplaceAll(res.Markdown, " ", ""), "HelloPDF") {
		t.Fatalf("expected extracted text to contain 'Hello PDF', got %q", res.Markdown)
	}
}

func TestPDFMalformedReturnsError(t *testing.T) {
	_, err := (PDF{}).Convert(strings.NewReader("%PDF-1.4 not really a pdf"), convert.StreamInfo{Extension: ".pdf"})
	if err == nil {
		t.Fatal("expected an error (or recovered panic) on malformed PDF")
	}
}
