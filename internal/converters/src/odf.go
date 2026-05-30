package converters

import (
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// ODF converts ODT/ODS/ODP files by reading content.xml, mapping text:h
// outline-level → #..######, text:p → paragraphs (or list items inside
// text:list), and table:table → Markdown tables.
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
	return convert.Result{Markdown: odfToMarkdown(data)}, nil
}

func odfToMarkdown(data []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var sb strings.Builder
	listDepth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "h":
				lvl := 1
				if v, err := strconv.Atoi(attrLocal(t, "outline-level")); err == nil && v > 0 {
					lvl = v
				}
				if lvl > 6 {
					lvl = 6
				}
				txt := strings.TrimSpace(readODFInline(dec, "h"))
				if txt != "" {
					sb.WriteString(strings.Repeat("#", lvl) + " " + txt + "\n\n")
				}
			case "p":
				txt := strings.TrimSpace(readODFInline(dec, "p"))
				if txt == "" {
					continue
				}
				if listDepth > 0 {
					sb.WriteString(strings.Repeat("  ", listDepth-1) + "- " + txt + "\n")
				} else {
					sb.WriteString(txt + "\n\n")
				}
			case "list":
				listDepth++
			case "table":
				if rows := readODFTable(dec); len(rows) > 0 {
					sb.WriteString(toMarkdownTable(rows))
					sb.WriteString("\n")
				}
			}
		case xml.EndElement:
			if t.Name.Local == "list" && listDepth > 0 {
				listDepth--
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// readODFInline reads character data inside <text:p>/<text:h> until the
// matching end tag, treating tab/line-break as their characters and ignoring
// inline formatting elements (text:span etc).
func readODFInline(dec *xml.Decoder, elem string) string {
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return b.String()
		}
		switch t := tok.(type) {
		case xml.CharData:
			b.Write(t)
		case xml.StartElement:
			switch t.Name.Local {
			case "tab":
				b.WriteByte('\t')
			case "line-break":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			if t.Name.Local == elem {
				return b.String()
			}
		}
	}
}

func readODFTable(dec *xml.Decoder) [][]string {
	var rows [][]string
	for {
		tok, err := dec.Token()
		if err != nil {
			return rows
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "table-row" {
				if cells := readODFTableRow(dec); len(cells) > 0 {
					rows = append(rows, cells)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "table" {
				return rows
			}
		}
	}
}

func readODFTableRow(dec *xml.Decoder) []string {
	var cells []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return cells
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "table-cell" {
				cells = append(cells, readODFTableCell(dec))
			}
		case xml.EndElement:
			if t.Name.Local == "table-row" {
				return cells
			}
		}
	}
}

func readODFTableCell(dec *xml.Decoder) string {
	var parts []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return strings.Join(parts, " ")
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "p" {
				if s := strings.TrimSpace(readODFInline(dec, "p")); s != "" {
					parts = append(parts, s)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "table-cell" {
				return strings.Join(parts, " ")
			}
		}
	}
}
