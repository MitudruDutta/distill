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

func mustConvertPDFium(t *testing.T, g *fpdf.Fpdf) string {
	t.Helper()
	var buf bytes.Buffer
	if err := g.Output(&buf); err != nil {
		t.Fatal(err)
	}
	res, err := (PDFium{}).Convert(&buf, convert.StreamInfo{Extension: ".pdf"})
	if err != nil {
		t.Fatal(err)
	}
	return res.Markdown
}

func TestPDFiumExtractsText(t *testing.T) {
	g := fpdf.New("P", "pt", "Letter", "")
	g.SetCompression(false)
	g.AddPage()
	g.SetFont("Helvetica", "", 24)
	g.Cell(200, 30, "Hello PDFium")
	got := mustConvertPDFium(t, g)
	if !strings.Contains(strings.ReplaceAll(got, " ", ""), "HelloPDFium") {
		t.Fatalf("got %q", got)
	}
}

func TestPDFiumReconstructsTable(t *testing.T) {
	g := fpdf.New("P", "pt", "Letter", "")
	g.SetCompression(false)
	g.AddPage()
	g.SetFont("Helvetica", "", 12)
	for _, c := range []struct {
		x, y float64
		s    string
	}{{72, 72, "Name"}, {300, 72, "Age"}, {72, 100, "Ada"}, {300, 100, "36"}, {72, 128, "Bob"}, {300, 128, "41"}} {
		g.SetXY(c.x, c.y)
		g.Cell(120, 14, c.s)
	}
	got := mustConvertPDFium(t, g)
	for _, want := range []string{"| Name | Age |", "| --- | --- |", "| Ada | 36 |", "| Bob | 41 |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in reconstructed table:\n%s", want, got)
		}
	}
}

func TestPDFiumProseIsNotMisreadAsTable(t *testing.T) {
	g := fpdf.New("P", "pt", "Letter", "")
	g.SetCompression(false)
	g.AddPage()
	g.SetFont("Helvetica", "", 12)
	g.SetXY(72, 72)
	g.MultiCell(450, 16, "This is ordinary prose that should not become a table at all.", "", "", false)
	got := mustConvertPDFium(t, g)
	if strings.Contains(got, "| --- |") {
		t.Fatalf("prose misdetected as a table:\n%s", got)
	}
	if !strings.Contains(got, "prose") {
		t.Fatalf("prose text missing:\n%s", got)
	}
}
