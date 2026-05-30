//go:build pdfium

package converters

import (
	"errors"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	gopdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/webassembly"

	"github.com/MitudruDutta/distill/internal/convert"
)

// PDFium extracts text using the bundled PDFium engine compiled to WebAssembly
// (pure-Go via wazero; no cgo). It detects headings via font size, bullets via
// leading whitespace, and reconstructs Markdown tables from column layout.
// Built with -tags pdfium it registers ahead of the default PDF converter and
// falls through to it on error/empty.
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
		page := requests.Page{ByIndex: &requests.PageByIndex{Document: doc.Document, Index: i}}
		var pageMD string
		if st, e := inst.GetPageTextStructured(&requests.GetPageTextStructured{
			Page:                   page,
			Mode:                   requests.GetPageTextStructuredModeRects,
			CollectFontInformation: true,
		}); e == nil {
			pageMD = reconstructPage(st.Rects)
		}
		if strings.TrimSpace(pageMD) == "" {
			if t, e := inst.GetPageText(&requests.GetPageText{Page: page}); e == nil {
				pageMD = strings.TrimSpace(t.Text)
			}
		}
		if pageMD != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(pageMD)
		}
	}
	if b.Len() == 0 {
		return convert.Result{}, errors.New("pdfium: no extractable text")
	}
	return convert.Result{Markdown: b.String()}, nil
}

// pdfBox holds one PDFium text rect plus the metadata needed for layout heuristics.
type pdfBox struct {
	text          string  // trimmed
	raw           string  // original (preserves leading space → bullet marker)
	left, top, bot float64
	height        float64
	fontSize      float64
}

// reconstructPage groups text rects into rows (Y) and decides whether the page
// is a table (rendered as Markdown table) or prose. For prose it emits headings
// via font-size ratio and bullets via leading whitespace.
func reconstructPage(rects []*responses.GetPageTextStructuredRect) string {
	var boxes []pdfBox
	for _, r := range rects {
		s := strings.TrimSpace(r.Text)
		if s == "" {
			continue
		}
		p := r.PointPosition
		size := 0.0
		if r.FontInformation != nil {
			size = r.FontInformation.Size
		}
		boxes = append(boxes, pdfBox{
			text:     s,
			raw:      r.Text,
			left:     p.Left,
			top:      p.Top,
			bot:      p.Bottom,
			height:   p.Top - p.Bottom,
			fontSize: size,
		})
	}
	if len(boxes) == 0 {
		return ""
	}

	heights := make([]float64, len(boxes))
	for i, b := range boxes {
		heights[i] = b.height
	}
	sort.Float64s(heights)
	medH := heights[len(heights)/2]
	if medH <= 0 {
		medH = 10
	}
	rowTol := medH * 0.7
	colTol := medH
	if colTol < 6 {
		colTol = 6
	}

	sort.Slice(boxes, func(i, j int) bool { return boxes[i].top > boxes[j].top })
	var rows [][]pdfBox
	for _, b := range boxes {
		if len(rows) == 0 || rows[len(rows)-1][0].top-b.top > rowTol {
			rows = append(rows, []pdfBox{b})
		} else {
			rows[len(rows)-1] = append(rows[len(rows)-1], b)
		}
	}
	for _, row := range rows {
		sort.Slice(row, func(i, j int) bool { return row[i].left < row[j].left })
	}

	asText := func() string { return renderProse(rows, boxes) }

	// Candidate columns from clustered left positions.
	lefts := make([]float64, len(boxes))
	for i, b := range boxes {
		lefts[i] = b.left
	}
	sort.Float64s(lefts)
	var cols []float64
	for _, x := range lefts {
		if len(cols) == 0 || x-cols[len(cols)-1] > colTol {
			cols = append(cols, x)
		}
	}
	near := func(x float64, cs []float64) int {
		best, bd := 0, math.Abs(x-cs[0])
		for i := 1; i < len(cs); i++ {
			if d := math.Abs(x - cs[i]); d < bd {
				best, bd = i, d
			}
		}
		return best
	}

	// A real table has FEW columns whose x-positions recur across MANY rows.
	support := make([]int, len(cols))
	for _, row := range rows {
		seen := map[int]bool{}
		for _, b := range row {
			if ci := near(b.left, cols); math.Abs(b.left-cols[ci]) <= colTol {
				seen[ci] = true
			}
		}
		for ci := range seen {
			support[ci]++
		}
	}
	minSup := (len(rows) + 1) / 2
	if minSup < 3 {
		minSup = 3
	}
	var strong []float64
	for ci, c := range cols {
		if support[ci] >= minSup {
			strong = append(strong, c)
		}
	}
	if len(strong) < 2 || len(strong) > 6 || len(rows) < 3 {
		return asText()
	}
	multi := 0
	for _, row := range rows {
		seen := map[int]bool{}
		for _, b := range row {
			if ci := near(b.left, strong); math.Abs(b.left-strong[ci]) <= colTol*1.5 {
				seen[ci] = true
			}
		}
		if len(seen) >= 2 {
			multi++
		}
	}
	if float64(multi) < 0.7*float64(len(rows)) {
		return asText()
	}

	grid := make([][]string, len(rows))
	for ri, row := range rows {
		cells := make([]string, len(strong))
		for _, b := range row {
			ci := near(b.left, strong)
			if cells[ci] != "" {
				cells[ci] += " " + b.text
			} else {
				cells[ci] = b.text
			}
		}
		grid[ri] = cells
	}
	return strings.TrimRight(toMarkdownTable(grid), "\n")
}

// renderProse converts non-table rows into LLM-friendly Markdown: headings via
// font-size ratio, bullets via leading whitespace.
func renderProse(rows [][]pdfBox, boxes []pdfBox) string {
	// Median font size across all boxes (used to detect oversized = heading).
	var sizes []float64
	for _, b := range boxes {
		if b.fontSize > 0 {
			sizes = append(sizes, b.fontSize)
		}
	}
	var medFont float64
	if len(sizes) > 0 {
		sort.Float64s(sizes)
		medFont = sizes[len(sizes)/2]
	}

	var out []string
	blank := func() {
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
	}

	for _, row := range rows {
		parts := make([]string, len(row))
		for i, b := range row {
			parts[i] = b.text
		}
		text := strings.Join(parts, " ")

		// Bullet: detect either leading whitespace in raw text (pdftotext-style)
		// OR a bullet glyph as the first rune of any rect (PDFs commonly render
		// • via a private-use codepoint specific to the document's font).
		isBullet := false
		if r := row[0].raw; len(r) > 0 && (r[0] == ' ' || r[0] == '\t') {
			isBullet = true
		}
		if !isBullet {
			for _, b := range row {
				if first, _ := utf8.DecodeRuneInString(b.text); isBulletGlyph(first) {
					isBullet = true
					break
				}
			}
		}

		// Heading: largest font in the row, scaled against page median.
		rowSize := 0.0
		for _, b := range row {
			if b.fontSize > rowSize {
				rowSize = b.fontSize
			}
		}
		ratio := 0.0
		if medFont > 0 && rowSize > 0 {
			ratio = rowSize / medFont
		}

		switch {
		case ratio >= 1.5:
			blank()
			out = append(out, "# "+text)
			out = append(out, "")
		case ratio >= 1.25:
			blank()
			out = append(out, "## "+text)
			out = append(out, "")
		case ratio >= 1.12:
			blank()
			out = append(out, "### "+text)
			out = append(out, "")
		case isBullet:
			out = append(out, "- "+stripBulletPrefix(text))
		default:
			out = append(out, text)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// isBulletGlyph reports whether r is a codepoint commonly used as a bullet
// marker in PDFs. The Private Use Area (U+E000–U+F8FF) is treated as a bullet
// because most fonts use those codepoints for custom symbols, and resume/CV
// PDFs use them for bullets specifically.
func isBulletGlyph(r rune) bool {
	switch r {
	case '\u2022', '\u2023', '\u2043', '\u2219', // • ‣ ⁃ ∙
		'\u25AA', '\u25AB', '\u25CB', '\u25CF', '\u25E6', // ▪ ▫ ○ ● ◦
		'\u25C6', '\u25C7', '\u25BA', '\u25B8', '\u00B7': // ◆ ◇ ► ▸ ·
		return true
	}
	return r >= 0xE000 && r <= 0xF8FF
}

// stripBulletPrefix removes any leading whitespace and bullet glyphs.
func stripBulletPrefix(s string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || isBulletGlyph(r)
	})
}
