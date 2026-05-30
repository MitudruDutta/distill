package converters_test

import (
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestFeedRSSRendersTitleAndItems(t *testing.T) {
	in := `<rss version="2.0"><channel><title>Blog</title>` +
		`<item><title>One</title><link>http://x/1</link><pubDate>2026</pubDate></item>` +
		`<item><title>Two</title></item></channel></rss>`
	res, err := (Feed{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".rss"})
	if err != nil {
		t.Fatal(err)
	}
	want := "# Blog\n\n- [One](http://x/1) — 2026\n- Two\n"
	if res.Markdown != want {
		t.Fatalf("\n got: %q\nwant: %q", res.Markdown, want)
	}
	if res.Title != "Blog" {
		t.Errorf("title: got %q, want Blog", res.Title)
	}
}

func TestFeedAtomRendersTitleAndEntries(t *testing.T) {
	in := `<feed xmlns="http://www.w3.org/2005/Atom"><title>Site</title>` +
		`<entry><title>A</title><link href="http://y/a"/><updated>2026-01-01</updated></entry></feed>`
	res, err := (Feed{}).Convert(strings.NewReader(in), convert.StreamInfo{Extension: ".atom"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "# Site") || !strings.Contains(res.Markdown, "- [A](http://y/a) — 2026-01-01") {
		t.Fatalf("atom output:\n%s", res.Markdown)
	}
}

func TestFeedRejectsNonFeedXML(t *testing.T) {
	if _, err := (Feed{}).Convert(strings.NewReader(`<note><to>x</to></note>`), convert.StreamInfo{Extension: ".xml"}); err == nil {
		t.Fatal("want an error for non-feed XML, got nil")
	}
}
