package converters

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/MitudruDutta/distill/internal/convert"
)

// JSON validates and pretty-prints JSON into a fenced ```json block.
type JSON struct{}

func (JSON) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".json" || info.Mimetype == "application/json"
}

func (JSON) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: "```json\n" + buf.String() + "\n```"}, nil
}
