package converters

import (
	"bytes"
	"fmt"
	"io"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/ledongthuc/pdf"
)

// PDF extracts plain text from a PDF using a pure-Go reader (no cgo). Fidelity
// is best-effort: complex layouts, scanned (image-only) pages, and unusual
// fonts may extract poorly. Higher-fidelity extraction via PDFium and table
// reconstruction are planned behind a build tag (Phase 3B).
type PDF struct{}

func (PDF) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".pdf" || info.Mimetype == "application/pdf"
}

func (PDF) Convert(r io.Reader, _ convert.StreamInfo) (res convert.Result, err error) {
	// The pure-Go reader can panic on malformed input; treat that as an error
	// rather than letting it crash the process (input is untrusted).
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("distill: pdf parse failed: %v", p)
		}
	}()

	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	rd, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return convert.Result{}, err
	}
	tr, err := rd.GetPlainText()
	if err != nil {
		return convert.Result{}, err
	}
	text, err := io.ReadAll(tr)
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: string(text)}, nil
}
