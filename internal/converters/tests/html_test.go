package converters_test

import (
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestHTMLConvertsHeadingsLinksAndLists(t *testing.T) {
	res, err := (HTML{}).Convert(strings.NewReader(
		`<h1>Hi</h1><p>a <b>bold</b> <a href="http://e">link</a></p><ul><li>one</li><li>two</li></ul>`),
		convert.StreamInfo{Mimetype: "text/html"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Hi", "**bold**", "[link](http://e)", "one", "two"} {
		if !strings.Contains(res.Markdown, want) {
			t.Errorf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}

// Security/fidelity: <script> and <style> contents must never leak into output.
func TestHTMLDropsScriptAndStyleContent(t *testing.T) {
	res, err := (HTML{}).Convert(strings.NewReader(
		`<p>safe</p><script>alert('xss')</script><style>.x{color:red}</style>`),
		convert.StreamInfo{Mimetype: "text/html"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Markdown, "alert") || strings.Contains(res.Markdown, "color:red") {
		t.Fatalf("script/style content leaked into output:\n%s", res.Markdown)
	}
}
