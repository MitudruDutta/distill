package converters

import (
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// DOCX converts a Word document to Markdown, preserving heading levels
// (Heading1..N → # ..######), bullet lists (numPr → -), and tables (w:tbl).
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
	return convert.Result{Markdown: docxToMarkdown(data)}, nil
}

func docxToMarkdown(data []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var sb strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "p":
			renderDocxP(dec, &sb)
		case "tbl":
			renderDocxTbl(dec, &sb)
		}
	}
	return strings.TrimSpace(sb.String())
}

// renderDocxP processes a <w:p> until </w:p>, detecting heading style or list
// numbering and writing Markdown output.
func renderDocxP(dec *xml.Decoder, sb *strings.Builder) {
	heading := 0
	listLvl := -1
	var text strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "pStyle":
				if h := headingFromStyle(attrLocal(t, "val")); h > 0 {
					heading = h
				}
			case "ilvl":
				if v, err := strconv.Atoi(attrLocal(t, "val")); err == nil {
					listLvl = v
				} else if listLvl < 0 {
					listLvl = 0
				}
			case "numId":
				if listLvl < 0 {
					listLvl = 0
				}
			case "t":
				var s string
				if err := dec.DecodeElement(&s, &t); err == nil {
					text.WriteString(s)
				}
			case "tab":
				text.WriteByte('\t')
			case "br":
				text.WriteByte('\n')
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				txt := strings.TrimSpace(text.String())
				switch {
				case heading > 0 && txt != "":
					sb.WriteString(strings.Repeat("#", heading) + " " + txt + "\n\n")
				case listLvl >= 0 && txt != "":
					sb.WriteString(strings.Repeat("  ", listLvl) + "- " + txt + "\n")
				case txt != "":
					sb.WriteString(txt + "\n\n")
				}
				return
			}
		}
	}
}

func renderDocxTbl(dec *xml.Decoder, sb *strings.Builder) {
	var rows [][]string
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "tr" {
				if cells := readDocxTr(dec); len(cells) > 0 {
					rows = append(rows, cells)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "tbl" {
				if len(rows) > 0 {
					sb.WriteString(toMarkdownTable(rows))
					sb.WriteString("\n")
				}
				return
			}
		}
	}
}

func readDocxTr(dec *xml.Decoder) []string {
	var cells []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return cells
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "tc" {
				cells = append(cells, readDocxTc(dec))
			}
		case xml.EndElement:
			if t.Name.Local == "tr" {
				return cells
			}
		}
	}
}

func readDocxTc(dec *xml.Decoder) string {
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return strings.TrimSpace(b.String())
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				var s string
				if err := dec.DecodeElement(&s, &t); err == nil {
					if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
						b.WriteByte(' ')
					}
					b.WriteString(s)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "tc" {
				return strings.TrimSpace(b.String())
			}
		}
	}
}

// headingFromStyle maps a Word style id to a Markdown heading level (1..6) or
// returns 0 if it isn't a heading.
func headingFromStyle(val string) int {
	v := strings.ToLower(val)
	switch v {
	case "title":
		return 1
	case "subtitle":
		return 2
	}
	if strings.HasPrefix(v, "heading") {
		if n, err := strconv.Atoi(strings.TrimPrefix(v, "heading")); err == nil && n >= 1 {
			if n > 6 {
				return 6
			}
			return n
		}
	}
	return 0
}

// attrLocal returns the value of the attribute whose local name matches.
func attrLocal(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}
