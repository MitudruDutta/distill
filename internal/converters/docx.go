package converters

import (
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// DOCX extracts paragraph text from a Word document's word/document.xml.
// Headings and tables are currently flattened to paragraphs (text preserved).
type DOCX struct{}

func (DOCX) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".docx" ||
		info.Mimetype == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

func (DOCX) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	zr, err := openZip(r)
	if err != nil {
		return convert.Result{}, err
	}
	data, err := zipEntry(zr, "word/document.xml")
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: strings.Join(extractParagraphs(data, "p"), "\n\n")}, nil
}
