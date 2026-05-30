package converters_test

import (
	"bytes"
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestArchiveZipConvertsEachEntry(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"a.csv": "x,y\n1,2\n",
		"b.txt": "hello world",
	})
	res, err := (Archive{Reg: Default()}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".zip"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## a.csv", "| x | y |", "## b.txt", "hello world"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}
