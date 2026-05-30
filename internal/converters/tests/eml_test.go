package converters_test

import (
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestEMLMultipartPrefersPlainText(t *testing.T) {
	raw := "Subject: Greetings\r\n" +
		"Content-Type: multipart/alternative; boundary=BB\r\n\r\n" +
		"--BB\r\nContent-Type: text/plain\r\n\r\nplain hello\r\n" +
		"--BB\r\nContent-Type: text/html\r\n\r\n<p>html hello</p>\r\n--BB--\r\n"
	res, err := (EML{}).Convert(strings.NewReader(raw), convert.StreamInfo{Extension: ".eml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "**Subject:** Greetings") || !strings.Contains(res.Markdown, "plain hello") {
		t.Fatalf("got:\n%s", res.Markdown)
	}
	if strings.Contains(res.Markdown, "html hello") {
		t.Fatal("should prefer text/plain over text/html")
	}
}

func TestEMLHTMLBodyConvertedToMarkdown(t *testing.T) {
	raw := "Subject: H\r\nContent-Type: text/html\r\n\r\n<h1>Title</h1><p>Body</p>"
	res, err := (EML{}).Convert(strings.NewReader(raw), convert.StreamInfo{Extension: ".eml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# Title") {
		t.Fatalf("expected converted HTML, got:\n%s", res.Markdown)
	}
}
