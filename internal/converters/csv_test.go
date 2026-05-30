package converters

import (
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestCSVTable(t *testing.T) {
	in := "name,age\nAda,36\n\"Smith, J\",40\n"
	res, err := (CSV{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".csv"})
	if err != nil {
		t.Fatal(err)
	}
	want := "| name | age |\n| --- | --- |\n| Ada | 36 |\n| Smith, J | 40 |\n"
	if res.Markdown != want {
		t.Fatalf("csv table mismatch:\n got: %q\nwant: %q", res.Markdown, want)
	}
}

func TestCSVAccepts(t *testing.T) {
	if !(CSV{}).Accepts(convert.StreamInfo{Extension: ".csv"}) {
		t.Fatal("should accept .csv")
	}
	if (CSV{}).Accepts(convert.StreamInfo{Extension: ".txt"}) {
		t.Fatal("should not accept .txt")
	}
}
