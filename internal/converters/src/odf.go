package converters

import (
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// ODF extracts text from OpenDocument files (.odt/.ods/.odp) by reading the
// paragraphs and headings of content.xml.
type ODF struct{}

var odfExts = map[string]bool{".odt": true, ".ods": true, ".odp": true}

func (ODF) Accepts(info convert.StreamInfo) bool {
	return odfExts[info.Extension] ||
		strings.HasPrefix(info.Mimetype, "application/vnd.oasis.opendocument.")
}

func (ODF) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	zr, err := openZip(r)
	if err != nil {
		return convert.Result{}, err
	}
	data, err := zipEntry(zr, "content.xml")
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: strings.Join(extractParagraphs(data, "p", "h"), "\n\n")}, nil
}
