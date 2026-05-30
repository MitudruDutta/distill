package converters

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// PPTX extracts text from each slide of a PowerPoint file, in slide order.
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

	var names []string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			names = append(names, f.Name)
		}
	}
	sort.Slice(names, func(i, j int) bool { return slideNum(names[i]) < slideNum(names[j]) })

	var b strings.Builder
	for i, name := range names {
		data, err := zipEntry(zr, name)
		if err != nil {
			continue
		}
		paras := extractParagraphs(data, "p")
		if len(paras) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## Slide %d\n\n%s\n\n", i+1, strings.Join(paras, "\n\n"))
	}
	return convert.Result{Markdown: strings.TrimSpace(b.String())}, nil
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
