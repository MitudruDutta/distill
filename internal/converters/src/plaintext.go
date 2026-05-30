package converters

import (
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// PlainText is the generic catch-all converter (priority 10). It treats the
// input as text and passes it through unchanged.
type PlainText struct{}

func (PlainText) Accepts(info convert.StreamInfo) bool {
	if strings.HasPrefix(info.Mimetype, "text/") {
		return true
	}
	// Catch-all when there is no stronger signal.
	return info.Mimetype == "" || info.Mimetype == "application/octet-stream"
}

func (PlainText) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: string(b)}, nil
}
