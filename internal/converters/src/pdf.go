package converters

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/ledongthuc/pdf"
)

// PDF extracts text from a PDF. It prefers poppler's `pdftotext -layout` when
// available (better spacing/layout), falls back to a pure-Go reader, and as a
// last resort OCRs scanned PDFs when Tesseract + pdftoppm are present.
type PDF struct{}

func (PDF) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".pdf" || info.Mimetype == "application/pdf"
}

func (PDF) Convert(r io.Reader, _ convert.StreamInfo) (res convert.Result, err error) {
	// The pure-Go reader can panic on malformed input; treat that as an error.
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("distill: pdf parse failed: %v", p)
		}
	}()

	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}

	if toolAvailable("pdftotext") {
		if out, e := runToolStdin("pdftotext", data, "-layout", "-", "-"); e == nil {
			if s := strings.TrimSpace(string(out)); s != "" {
				return convert.Result{Markdown: s}, nil
			}
		}
	}

	text, e := pureGoPDFText(data)
	if e == nil && strings.TrimSpace(text) != "" {
		return convert.Result{Markdown: strings.TrimSpace(text)}, nil
	}

	if ocrAvailable() && toolAvailable("pdftoppm") {
		if t, e2 := ocrPDFText(data); e2 == nil && t != "" {
			return convert.Result{Markdown: t}, nil
		}
	}

	if e != nil {
		return convert.Result{}, e
	}
	return convert.Result{Markdown: strings.TrimSpace(text)}, nil
}

func pureGoPDFText(data []byte) (string, error) {
	rd, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	tr, err := rd.GetPlainText()
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(tr)
	return string(b), err
}
