package converters

import (
	"io"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/MitudruDutta/distill/internal/convert"
)

// HTML converts HTML to Markdown via html-to-markdown/v2 (CommonMark output
// with ATX headings; scripts and styles are dropped by the library).
type HTML struct{}

func (HTML) Accepts(info convert.StreamInfo) bool {
	switch info.Extension {
	case ".html", ".htm":
		return true
	}
	return strings.HasPrefix(info.Mimetype, "text/html")
}

func (HTML) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	md, err := htmltomarkdown.ConvertReader(r)
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: string(md)}, nil
}
