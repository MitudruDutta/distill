package converters

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// PPTX converts a PowerPoint deck to Markdown: each slide becomes "## Slide N"
// (or "## Slide N: <title>" with a title placeholder), with body text emitted
// as bullets honoring lvl, and embedded tables (a:tbl in a graphicFrame)
// rendered as Markdown tables.
type PPTX struct{}

func (PPTX) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".pptx" ||
		info.Mimetype == "application/vnd.openxmlformats-officedocument.presentationml.presentation"
}

func (PPTX) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	zr, err := openZip(r)
	if err != nil {
		return convert.Result{}, err
	}
	var slides []string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slides = append(slides, f.Name)
		}
	}
	sort.Slice(slides, func(i, j int) bool { return slideNum(slides[i]) < slideNum(slides[j]) })

	var sb strings.Builder
	for i, name := range slides {
		data, err := zipEntry(zr, name)
		if err != nil {
			continue
		}
		if md := pptxSlideToMarkdown(i+1, data); md != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(md)
		}
	}
	return convert.Result{Markdown: strings.TrimSpace(sb.String())}, nil
}

type pptxPara struct {
	level int
	text  string
}

type pptxBlock struct {
	kind  string // "para", "bullet", "table"
	text  string
	level int
	rows  [][]string
}

func pptxSlideToMarkdown(num int, data []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var title string
	var blocks []pptxBlock

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
		case "sp":
			isTitle, paras := readPPTXShape(dec)
			for _, p := range paras {
				if p.text == "" {
					continue
				}
				switch {
				case isTitle && title == "":
					title = p.text
				case isTitle:
					blocks = append(blocks, pptxBlock{kind: "para", text: p.text})
				default:
					blocks = append(blocks, pptxBlock{kind: "bullet", text: p.text, level: p.level})
				}
			}
		case "tbl":
			if rows := readPPTXTable(dec); len(rows) > 0 {
				blocks = append(blocks, pptxBlock{kind: "table", rows: rows})
			}
		case "p":
			// Tolerate top-level <a:p> outside any shape (used by simple fixtures).
			if p := readPPTXPara(dec); p.text != "" {
				blocks = append(blocks, pptxBlock{kind: "bullet", text: p.text, level: p.level})
			}
		}
	}

	var sb strings.Builder
	if title != "" {
		fmt.Fprintf(&sb, "## Slide %d: %s\n\n", num, title)
	} else {
		fmt.Fprintf(&sb, "## Slide %d\n\n", num)
	}
	for _, b := range blocks {
		switch b.kind {
		case "para":
			sb.WriteString(b.text + "\n\n")
		case "bullet":
			sb.WriteString(strings.Repeat("  ", b.level) + "- " + b.text + "\n")
		case "table":
			sb.WriteString(toMarkdownTable(b.rows))
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

// readPPTXShape walks a <p:sp>, returning whether it carries a title
// placeholder and the paragraphs of text inside its <p:txBody>.
func readPPTXShape(dec *xml.Decoder) (bool, []pptxPara) {
	isTitle := false
	var paras []pptxPara
	for {
		tok, err := dec.Token()
		if err != nil {
			return isTitle, paras
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "ph":
				phType := strings.ToLower(attrLocal(t, "type"))
				if phType == "title" || phType == "ctrtitle" {
					isTitle = true
				}
			case "p":
				paras = append(paras, readPPTXPara(dec))
			}
		case xml.EndElement:
			if t.Name.Local == "sp" {
				return isTitle, paras
			}
		}
	}
}

// readPPTXPara collects text from an <a:p> until </a:p>, capturing the
// optional <a:pPr lvl="N"/> indent level and treating <a:br/> as a newline.
func readPPTXPara(dec *xml.Decoder) pptxPara {
	level := 0
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "pPr":
				if v, err := strconv.Atoi(attrLocal(t, "lvl")); err == nil {
					level = v
				}
			case "t":
				var s string
				if err := dec.DecodeElement(&s, &t); err == nil {
					b.WriteString(s)
				}
			case "br":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				return pptxPara{level: level, text: strings.TrimSpace(b.String())}
			}
		}
	}
	return pptxPara{level: level, text: strings.TrimSpace(b.String())}
}

func readPPTXTable(dec *xml.Decoder) [][]string {
	var rows [][]string
	for {
		tok, err := dec.Token()
		if err != nil {
			return rows
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "tr" {
				if cells := readPPTXTableRow(dec); len(cells) > 0 {
					rows = append(rows, cells)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "tbl" {
				return rows
			}
		}
	}
}

func readPPTXTableRow(dec *xml.Decoder) []string {
	var cells []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return cells
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "tc" {
				cells = append(cells, readPPTXTableCell(dec))
			}
		case xml.EndElement:
			if t.Name.Local == "tr" {
				return cells
			}
		}
	}
}

func readPPTXTableCell(dec *xml.Decoder) string {
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

// slideNum extracts N from ".../slideN.xml" so slide10 sorts after slide2.
func slideNum(name string) int {
	base := name[strings.LastIndex(name, "/")+1:]
	base = strings.TrimSuffix(strings.TrimPrefix(base, "slide"), ".xml")
	n := 0
	for _, c := range base {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
