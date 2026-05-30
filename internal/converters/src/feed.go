package converters

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// Feed converts an RSS or Atom feed into a title heading plus a list of dated,
// linked items. Non-feed input returns errNotFeed so the dispatcher falls
// through to the generic XML converter.
type Feed struct{}

var errNotFeed = errors.New("distill: not an RSS or Atom feed")

func (Feed) Accepts(info convert.StreamInfo) bool {
	switch info.Extension {
	case ".rss", ".atom", ".xml":
		return true
	}
	switch info.Mimetype {
	case "application/rss+xml", "application/atom+xml":
		return true
	}
	return false
}

func (Feed) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}

	var rss struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				Title string `xml:"title"`
				Link  string `xml:"link"`
				Date  string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if xml.Unmarshal(data, &rss) == nil && rss.XMLName.Local == "rss" {
		var b strings.Builder
		writeFeedHeader(&b, rss.Channel.Title)
		for _, it := range rss.Channel.Items {
			writeFeedItem(&b, it.Title, it.Link, it.Date)
		}
		return convert.Result{Markdown: b.String(), Title: rss.Channel.Title}, nil
	}

	var atom struct {
		XMLName xml.Name `xml:"feed"`
		Title   string   `xml:"title"`
		Entries []struct {
			Title string `xml:"title"`
			Link  struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
			Date string `xml:"updated"`
		} `xml:"entry"`
	}
	if xml.Unmarshal(data, &atom) == nil && atom.XMLName.Local == "feed" {
		var b strings.Builder
		writeFeedHeader(&b, atom.Title)
		for _, e := range atom.Entries {
			writeFeedItem(&b, e.Title, e.Link.Href, e.Date)
		}
		return convert.Result{Markdown: b.String(), Title: atom.Title}, nil
	}

	return convert.Result{}, errNotFeed
}

func writeFeedHeader(b *strings.Builder, title string) {
	if t := strings.TrimSpace(title); t != "" {
		b.WriteString("# ");b.WriteString(t);b.WriteString("\n\n")
	}
}

func writeFeedItem(b *strings.Builder, title, link, date string) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "(untitled)"
	}
	if link = strings.TrimSpace(link); link != "" {
		b.WriteString("- [");b.WriteString(title);b.WriteString("](");b.WriteString(link);b.WriteString(")")
	} else {
		b.WriteString("- ");b.WriteString(title)
	}
	if d := strings.TrimSpace(date); d != "" {
		b.WriteString(" — ");b.WriteString(d)
	}
	b.WriteByte('\n')
}
