//go:build pdfium

package converters

import (
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	gopdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"

	"github.com/MitudruDutta/distill/internal/convert"
)

// PDFium extracts text using the bundled PDFium engine compiled to WebAssembly
// (pure-Go via wazero; no cgo). Built with -tags pdfium it registers ahead of
// the default PDF converter and falls through to it on error/empty (so scanned
// PDFs still reach the OCR path).
type PDFium struct{}

func registerPDFium(reg *convert.Registry) { reg.Register(PDFium{}, -1) }

var (
	pdfiumPool gopdfium.Pool
	pdfiumOnce sync.Once
	pdfiumErr  error
)

func pdfiumPoolInstance() (gopdfium.Pool, error) {
	pdfiumOnce.Do(func() {
		pdfiumPool, pdfiumErr = webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1})
	})
	return pdfiumPool, pdfiumErr
}

func (PDFium) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".pdf" || info.Mimetype == "application/pdf"
}

func (PDFium) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	pool, err := pdfiumPoolInstance()
	if err != nil {
		return convert.Result{}, err
	}
	inst, err := pool.GetInstance(30 * time.Second)
	if err != nil {
		return convert.Result{}, err
	}
	defer inst.Close()

	doc, err := inst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return convert.Result{}, err
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pc, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return convert.Result{}, err
	}

	var b strings.Builder
	for i := 0; i < pc.PageCount; i++ {
		t, err := inst.GetPageText(&requests.GetPageText{
			Page: requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: i}},
		})
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(t.Text); s != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(s)
		}
	}
	if b.Len() == 0 {
		return convert.Result{}, errors.New("pdfium: no extractable text")
	}
	return convert.Result{Markdown: b.String()}, nil
}
