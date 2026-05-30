package converters

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// Image emits basic metadata (format and pixel dimensions) for common raster
// images. It reads only the header, not the pixels.
type Image struct{}

var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true}

func (Image) Accepts(info convert.StreamInfo) bool {
	return imageExts[info.Extension] || strings.HasPrefix(info.Mimetype, "image/")
}

func (Image) Convert(r io.Reader, info convert.StreamInfo) (convert.Result, error) {
	cfg, format, err := image.DecodeConfig(r)
	if err != nil {
		return convert.Result{}, err
	}
	name := info.Filename
	if name == "" {
		name = "image"
	}
	md := fmt.Sprintf("**%s** — %s, %d×%d px", name, format, cfg.Width, cfg.Height)
	return convert.Result{Markdown: md, Title: name}, nil
}
