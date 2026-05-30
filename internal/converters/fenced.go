package converters

import (
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// Fenced wraps input verbatim in a language-tagged code fence. Used for config
// and markup formats whose most faithful Markdown form is the original text
// (parsing then re-serializing would lose comments and ordering).
type Fenced struct {
	Lang  string
	Exts  []string
	Mimes []string
}

func (f Fenced) Accepts(info convert.StreamInfo) bool {
	for _, e := range f.Exts {
		if info.Extension == e {
			return true
		}
	}
	for _, m := range f.Mimes {
		if info.Mimetype == m {
			return true
		}
	}
	return false
}

func (f Fenced) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	content := strings.TrimRight(string(b), "\n")
	return convert.Result{Markdown: "```" + f.Lang + "\n" + content + "\n```"}, nil
}
