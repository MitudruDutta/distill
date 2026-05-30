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

	gopdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/webassembly"

	"github.com/MitudruDutta/distill/internal/convert"
)

// PDFium extracts text using the bundled PDFium engine compiled to WebAssembly
// (pure-Go via wazero; no cgo). It reconstructs Markdown tables from column
// layout. Built with -tags pdfium it registers ahead of the default PDF
// converter and falls through to it on error/empty.
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
			Page: page,
			Mode: requests.GetPageTextStructuredModeRects,
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

// reconstructPage groups text rects into rows (by vertical position) and columns
// (by left position). If the page is predominantly multi-column it is rendered
// as a Markdown table; otherwise as plain reading-order text.
func reconstructPage(rects []*responses.GetPageTextStructuredRect) string {
	type box struct {
		text      string
		left, top float64
		height    float64
	}
	var boxes []box
	for _, r := range rects {
		s := strings.TrimSpace(r.Text)
		if s == "" {
			continue
		}
		p := r.PointPosition
		boxes = append(boxes, box{s, p.Left, p.Top, p.Top - p.Bottom})
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

	// Rows: top-to-bottom (PDF y increases upward), grouped by top proximity.
	sort.Slice(boxes, func(i, j int) bool { return boxes[i].top > boxes[j].top })
	var rows [][]box
	for _, b := range boxes {
		if len(rows) == 0 || rows[len(rows)-1][0].top-b.top > rowTol {
			rows = append(rows, []box{b})
		} else {
			rows[len(rows)-1] = append(rows[len(rows)-1], b)
		}
	}
	for _, row := range rows {
		sort.Slice(row, func(i, j int) bool { return row[i].left < row[j].left })
	}

	asText := func() string {
		lines := make([]string, len(rows))
		for ri, row := range rows {
			parts := make([]string, len(row))
			for i, b := range row {
				parts[i] = b.text
			}
			lines[ri] = strings.Join(parts, " ")
		}
		return strings.Join(lines, "\n")
	}

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
	// Keep only columns supported by >=50% of rows (and >=3 rows); this rejects
	// prose/resumes that merely scatter text across horizontal positions.
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
