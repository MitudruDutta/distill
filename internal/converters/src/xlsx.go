package converters

import (
	"fmt"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/xuri/excelize/v2"
)

// XLSX renders each worksheet of an .xlsx workbook as a Markdown table under a
// heading with the sheet name.
type XLSX struct{}

func (XLSX) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".xlsx" ||
		info.Mimetype == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

func (XLSX) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return convert.Result{}, err
	}
	defer f.Close()

	var b strings.Builder
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return convert.Result{}, err
		}
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", sheet, toMarkdownTable(rows))
	}
	return convert.Result{Markdown: strings.TrimSpace(b.String())}, nil
}
