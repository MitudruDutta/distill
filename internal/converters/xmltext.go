package converters

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// extractParagraphs streams the XML in data and returns the trimmed text of
// each element whose local name is one of paraLocals (e.g. "p" for <w:p>/<a:p>,
// or "p" and "h" for ODF <text:p>/<text:h>). All character data inside such an
// element is concatenated, which captures <w:t>/<a:t>/<text:span> text without
// format-specific run handling. Empty paragraphs are dropped.
func extractParagraphs(data []byte, paraLocals ...string) []string {
	want := make(map[string]bool, len(paraLocals))
	for _, n := range paraLocals {
		want[n] = true
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	var (
		out   []string
		buf   strings.Builder
		depth int
	)
	for {
		tok, err := dec.Token()
		if err != nil {
			break // io.EOF or malformed tail: return what we have
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if want[t.Name.Local] {
				if depth == 0 {
					buf.Reset()
				}
				depth++
			}
		case xml.CharData:
			if depth > 0 {
				buf.Write(t)
			}
		case xml.EndElement:
			if want[t.Name.Local] && depth > 0 {
				depth--
				if depth == 0 {
					if s := strings.TrimSpace(buf.String()); s != "" {
						out = append(out, s)
					}
				}
			}
		}
	}
	return out
}
