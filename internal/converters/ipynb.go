package converters

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// Ipynb converts a Jupyter notebook (.ipynb) to Markdown: markdown cells are
// emitted verbatim, code cells as fenced blocks, and text outputs included.
type Ipynb struct{}

func (Ipynb) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".ipynb"
}

func (Ipynb) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	var nb struct {
		Cells []struct {
			CellType string          `json:"cell_type"`
			Source   json.RawMessage `json:"source"`
			Outputs  []struct {
				Text json.RawMessage            `json:"text"`
				Data map[string]json.RawMessage `json:"data"`
			} `json:"outputs"`
		} `json:"cells"`
		Metadata struct {
			LanguageInfo struct {
				Name string `json:"name"`
			} `json:"language_info"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(r).Decode(&nb); err != nil {
		return convert.Result{}, err
	}

	lang := nb.Metadata.LanguageInfo.Name
	if lang == "" {
		lang = "python"
	}

	var b strings.Builder
	for _, c := range nb.Cells {
		src := joinLines(c.Source)
		switch c.CellType {
		case "markdown":
			b.WriteString(src + "\n\n")
		case "code":
			if strings.TrimSpace(src) != "" {
				b.WriteString("```" + lang + "\n" + src + "\n```\n\n")
			}
			for _, o := range c.Outputs {
				txt := joinLines(o.Text)
				if strings.TrimSpace(txt) == "" {
					txt = joinLines(o.Data["text/plain"])
				}
				if strings.TrimSpace(txt) != "" {
					b.WriteString("```\n" + txt + "\n```\n\n")
				}
			}
		}
	}
	return convert.Result{Markdown: b.String()}, nil
}

// joinLines handles the ipynb convention where source/text may be either a
// single string or an array of line strings.
func joinLines(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var arr []string
	if json.Unmarshal(raw, &arr) == nil {
		return strings.Join(arr, "")
	}
	return ""
}
