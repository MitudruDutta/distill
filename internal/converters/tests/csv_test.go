package converters_test

import (
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestCSVRendersMarkdownTable(t *testing.T) {
	cases := []struct{ name, in, ext, want string }{
		{
			"basic with quoted comma",
			"name,age\nAda,36\n\"Smith, J\",40\n", ".csv",
			"| name | age |\n| --- | --- |\n| Ada | 36 |\n| Smith, J | 40 |\n",
		},
		{
			"pipe is escaped",
			"a|b,c\n1,2\n", ".csv",
			"| a\\|b | c |\n| --- | --- |\n| 1 | 2 |\n",
		},
		{
			"embedded newline becomes <br>",
			"h1,h2\n\"line1\nline2\",x\n", ".csv",
			"| h1 | h2 |\n| --- | --- |\n| line1<br>line2 | x |\n",
		},
		{
			"ragged rows pad to widest",
			"a,b,c\n1\n2,3\n", ".csv",
			"| a | b | c |\n| --- | --- | --- |\n| 1 |  |  |\n| 2 | 3 |  |\n",
		},
		{
			"tsv via extension",
			"a\tb\n1\t2\n", ".tsv",
			"| a | b |\n| --- | --- |\n| 1 | 2 |\n",
		},
		{
			"header only",
			"only,header\n", ".csv",
			"| only | header |\n| --- | --- |\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := (CSV{}).Convert(strings.NewReader(c.in), convert.StreamInfo{Extension: c.ext})
			if err != nil {
				t.Fatal(err)
			}
			if res.Markdown != c.want {
				t.Fatalf("\n got: %q\nwant: %q", res.Markdown, c.want)
			}
		})
	}
}

func TestCSVEmptyInputYieldsEmpty(t *testing.T) {
	res, err := (CSV{}).Convert(strings.NewReader(""), convert.StreamInfo{Extension: ".csv"})
	if err != nil || res.Markdown != "" {
		t.Fatalf("empty csv: got (%q, %v), want empty", res.Markdown, err)
	}
}

func TestCSVAcceptsCSVAndTSVNotTXT(t *testing.T) {
	for _, si := range []convert.StreamInfo{{Extension: ".csv"}, {Extension: ".tsv"}, {Mimetype: "text/csv"}} {
		if !(CSV{}).Accepts(si) {
			t.Errorf("should accept %+v", si)
		}
	}
	if (CSV{}).Accepts(convert.StreamInfo{Extension: ".txt"}) {
		t.Error("should not accept .txt")
	}
}
