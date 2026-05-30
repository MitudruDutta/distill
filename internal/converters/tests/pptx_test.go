package converters_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
	. "github.com/MitudruDutta/distill/internal/converters/src"
)

// pptxSlideXML builds a minimal slide with one optional title shape and one
// body shape carrying paragraphs (tuples of bullet level and text).
func pptxSlideXML(title string, body []struct {
	Lvl  int
	Text string
}) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><p:sld xmlns:a="urn:a" xmlns:p="urn:p"><p:cSld><p:spTree>`)
	if title != "" {
		b.WriteString(`<p:sp><p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:txBody>`)
		b.WriteString(`<a:p><a:r><a:t>` + title + `</a:t></a:r></a:p>`)
		b.WriteString(`</p:txBody></p:sp>`)
	}
	if len(body) > 0 {
		b.WriteString(`<p:sp><p:nvSpPr><p:nvPr/></p:nvSpPr><p:txBody>`)
		for _, p := range body {
			if p.Lvl > 0 {
				b.WriteString(`<a:p><a:pPr lvl="` + itoa(p.Lvl) + `"/><a:r><a:t>` + p.Text + `</a:t></a:r></a:p>`)
			} else {
				b.WriteString(`<a:p><a:r><a:t>` + p.Text + `</a:t></a:r></a:p>`)
			}
		}
		b.WriteString(`</p:txBody></p:sp>`)
	}
	b.WriteString(`</p:spTree></p:cSld></p:sld>`)
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [10]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestPPTXEmitsTitleAndSlideOrder(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"ppt/slides/slide1.xml": pptxSlideXML("First slide", nil),
		"ppt/slides/slide2.xml": pptxSlideXML("Second slide", nil),
	})
	res, err := (PPTX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".pptx"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Slide 1: First slide", "## Slide 2: Second slide"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
	if strings.Index(res.Markdown, "First") > strings.Index(res.Markdown, "Second") {
		t.Fatal("slides emitted out of order")
	}
}

func TestPPTXEmitsBulletsWithLevels(t *testing.T) {
	body := []struct {
		Lvl  int
		Text string
	}{{0, "Top point"}, {1, "Sub point"}, {0, "Another top"}}
	data := zipBytes(t, map[string]string{
		"ppt/slides/slide1.xml": pptxSlideXML("Topic", body),
	})
	res, _ := (PPTX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".pptx"})
	for _, want := range []string{"## Slide 1: Topic", "- Top point", "  - Sub point", "- Another top"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}

func TestPPTXExtractsEmbeddedTable(t *testing.T) {
	cell := func(s string) string {
		return `<a:tc><a:txBody><a:p><a:r><a:t>` + s + `</a:t></a:r></a:p></a:txBody></a:tc>`
	}
	row := func(a, b string) string { return `<a:tr>` + cell(a) + cell(b) + `</a:tr>` }
	slide := `<?xml version="1.0"?><p:sld xmlns:a="urn:a" xmlns:p="urn:p"><p:cSld><p:spTree>` +
		`<p:sp><p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:txBody>` +
		`<a:p><a:r><a:t>Fruit Data</a:t></a:r></a:p></p:txBody></p:sp>` +
		`<p:graphicFrame><a:graphic><a:graphicData><a:tbl>` +
		row("Fruit", "Qty") + row("Apple", "5") + row("Mango", "3") +
		`</a:tbl></a:graphicData></a:graphic></p:graphicFrame>` +
		`</p:spTree></p:cSld></p:sld>`
	data := zipBytes(t, map[string]string{"ppt/slides/slide1.xml": slide})
	res, _ := (PPTX{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".pptx"})
	for _, want := range []string{"## Slide 1: Fruit Data", "| Fruit | Qty |", "| --- | --- |", "| Apple | 5 |", "| Mango | 3 |"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}
