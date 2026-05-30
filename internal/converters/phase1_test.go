package converters

import (
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestJSONFenced(t *testing.T) {
	res, err := (JSON{}).Convert(strings.NewReader(`{"a":1,"b":[2,3]}`), convert.StreamInfo{Extension: ".json"})
	if err != nil {
		t.Fatal(err)
	}
	want := "```json\n{\n  \"a\": 1,\n  \"b\": [\n    2,\n    3\n  ]\n}\n```"
	if res.Markdown != want {
		t.Fatalf("json:\n got: %q\nwant: %q", res.Markdown, want)
	}
}

func TestFencedYAML(t *testing.T) {
	res, err := (Fenced{Lang: "yaml", Exts: []string{".yaml"}}).Convert(
		strings.NewReader("a: 1\n"), convert.StreamInfo{Extension: ".yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Markdown != "```yaml\na: 1\n```" {
		t.Fatalf("yaml fence: %q", res.Markdown)
	}
}

func TestFeedRSS(t *testing.T) {
	in := `<?xml version="1.0"?><rss version="2.0"><channel><title>My Blog</title>` +
		`<item><title>Post 1</title><link>http://x/1</link><pubDate>Mon, 01 Jan 2026</pubDate></item></channel></rss>`
	res, err := (Feed{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".rss"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# My Blog") || !strings.Contains(res.Markdown, "[Post 1](http://x/1)") {
		t.Fatalf("rss output:\n%s", res.Markdown)
	}
}

func TestFeedAtom(t *testing.T) {
	in := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Atom Site</title>` +
		`<entry><title>Entry A</title><link href="http://y/a"/><updated>2026-01-01</updated></entry></feed>`
	res, err := (Feed{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".atom"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# Atom Site") || !strings.Contains(res.Markdown, "[Entry A](http://y/a)") {
		t.Fatalf("atom output:\n%s", res.Markdown)
	}
}

func TestFeedNotAFeed(t *testing.T) {
	_, err := (Feed{}).Convert(strings.NewReader(`<note><to>x</to></note>`), convert.StreamInfo{Extension: ".xml"})
	if err != errNotFeed {
		t.Fatalf("want errNotFeed, got %v", err)
	}
}

func TestIpynb(t *testing.T) {
	in := `{"cells":[` +
		`{"cell_type":"markdown","source":["# Title\n","text"]},` +
		`{"cell_type":"code","source":"print(1)","outputs":[{"output_type":"stream","text":["1\n"]}]}` +
		`],"metadata":{"language_info":{"name":"python"}}}`
	res, err := (Ipynb{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".ipynb"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Title", "```python\nprint(1)\n```", "1"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("ipynb missing %q in:\n%s", want, res.Markdown)
		}
	}
}

func TestHTML(t *testing.T) {
	res, err := (HTML{}).Convert(strings.NewReader("<h1>Hi</h1><p>Yo</p>"), convert.StreamInfo{Mimetype: "text/html"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# Hi") || !strings.Contains(res.Markdown, "Yo") {
		t.Fatalf("html output: %q", res.Markdown)
	}
}

// TestRoutingHTMLBeatsPlainText verifies priority dispatch: HTML (5) wins over
// the plain-text catch-all (10) for text/html input.
func TestRoutingHTMLBeatsPlainText(t *testing.T) {
	res, err := Default().Convert(strings.NewReader("<h1>Hi</h1>"),
		[]convert.StreamInfo{{Extension: ".html", Mimetype: "text/html"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# Hi") {
		t.Fatalf("routing: expected HTML conversion, got %q", res.Markdown)
	}
}

// TestRoutingXMLFeedFallsThrough verifies a non-feed .xml falls from Feed (0)
// through to the generic XML fence (5).
func TestRoutingXMLFeedFallsThrough(t *testing.T) {
	res, err := Default().Convert(strings.NewReader("<note><to>x</to></note>"),
		[]convert.StreamInfo{{Extension: ".xml", Mimetype: "text/xml"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Markdown, "```xml") {
		t.Fatalf("routing: expected xml fence, got %q", res.Markdown)
	}
}
