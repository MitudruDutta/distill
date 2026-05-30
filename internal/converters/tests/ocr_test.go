package converters_test

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-pdf/fpdf"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/MitudruDutta/distill/internal/convert"
	. "github.com/MitudruDutta/distill/internal/converters/src"
)

func TestImageOCRRecoversRenderedText(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
	res, err := (Image{}).Convert(bytes.NewReader(renderTextPNG("DISTILL")),
		convert.StreamInfo{Extension: ".png", Filename: "t.png"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(res.Markdown), "DISTILL") {
		t.Fatalf("OCR did not recover the rendered text:\n%s", res.Markdown)
	}
}

func TestPDFScannedImageGetsOCR(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}
	// Build an image-only PDF (no text layer) so extraction falls back to OCR.
	gen := fpdf.New("P", "pt", "Letter", "")
	gen.AddPage()
	gen.RegisterImageOptionsReader("img", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(renderTextPNG("DISTILL")))
	gen.ImageOptions("img", 40, 40, 500, 0, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	var buf bytes.Buffer
	if err := gen.Output(&buf); err != nil {
		t.Fatal(err)
	}

	res, err := (PDF{}).Convert(&buf, convert.StreamInfo{Extension: ".pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(res.Markdown), "DISTILL") {
		t.Fatalf("scanned-PDF OCR did not recover text:\n%s", res.Markdown)
	}
}

// renderTextPNG draws msg in black on white using the stdlib bitmap font, then
// upscales it so Tesseract has large, high-contrast glyphs to read.
func renderTextPNG(msg string) []byte {
	small := image.NewRGBA(image.Rect(0, 0, 7*len(msg)+24, 24))
	draw.Draw(small, small.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	d := &font.Drawer{
		Dst:  small,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(10, 16),
	}
	d.DrawString(msg)

	big := image.NewRGBA(image.Rect(0, 0, small.Bounds().Dx()*8, small.Bounds().Dy()*8))
	xdraw.CatmullRom.Scale(big, big.Bounds(), small, small.Bounds(), xdraw.Over, nil)

	var buf bytes.Buffer
	_ = png.Encode(&buf, big)
	return buf.Bytes()
}
