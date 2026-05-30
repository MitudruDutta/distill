package converters_test

import (
	"bytes"
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestImageReportsFormatAndDimensions(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	res, err := (Image{}).Convert(&buf, convert.StreamInfo{Extension: ".png", Filename: "x.png"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"png", "3×2", "x.png"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in %q", want, res.Markdown)
		}
	}
}
