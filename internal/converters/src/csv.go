package converters

import (
	"encoding/csv"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// CSV converts CSV/TSV input into a Markdown table (priority 0).
type CSV struct{}

func (CSV) Accepts(info convert.StreamInfo) bool {
	switch info.Extension {
	case ".csv", ".tsv":
		return true
	}
	return info.Mimetype == "text/csv" || info.Mimetype == "text/tab-separated-values"
}

func (CSV) Convert(r io.Reader, info convert.StreamInfo) (convert.Result, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // allow ragged rows
	if info.Extension == ".tsv" || info.Mimetype == "text/tab-separated-values" {
		cr.Comma = '\t'
	}
	rows, err := cr.ReadAll()
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: toMarkdownTable(rows)}, nil
}

// cellReplacer makes a value safe for a single Markdown table cell: a raw
// newline would break the row, so it becomes <br>; pipes are escaped.
var cellReplacer = strings.NewReplacer("\r\n", "<br>", "\r", "<br>", "\n", "<br>", "|", "\\|")

// toMarkdownTable renders rows as a GitHub-flavored Markdown table. The first
// row is treated as the header.
func toMarkdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}

	var b strings.Builder
	writeRow := func(cells []string) {
		b.WriteByte('|')
		for c := 0; c < cols; c++ {
			cell := ""
			if c < len(cells) {
				cell = cellReplacer.Replace(cells[c])
			}
			b.WriteByte(' ')
			b.WriteString(cell)
			b.WriteString(" |")
		}
		b.WriteByte('\n')
	}

	writeRow(rows[0])
	b.WriteByte('|')
	for c := 0; c < cols; c++ {
		b.WriteString(" --- |")
	}
	b.WriteByte('\n')
	for _, row := range rows[1:] {
		writeRow(row)
	}
	return b.String()
}
