package converters

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// Image emits format + pixel dimensions for common raster images, and appends
// OCR text when Tesseract is available.
type Image struct{}

var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true}

func (Image) Accepts(info convert.StreamInfo) bool {
	return imageExts[info.Extension] || strings.HasPrefix(info.Mimetype, "image/")
}

func (Image) Convert(r io.Reader, info convert.StreamInfo) (convert.Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return convert.Result{}, err
	}
	name := info.Filename
	if name == "" {
		name = "image"
	}
	md := fmt.Sprintf("**%s** — %s, %d×%d px", name, format, cfg.Width, cfg.Height)
	if ocrAvailable() {
		if text, e := ocrImageText(data); e == nil && text != "" {
			md += "\n\n" + text
		}
	}
	return convert.Result{Markdown: md, Title: name}, nil
}
