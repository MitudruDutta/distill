package convert

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

type fakeConv struct {
	accept bool
	out    string
	err    error
}

func (f fakeConv) Accepts(StreamInfo) bool { return f.accept }
func (f fakeConv) Convert(r io.Reader, _ StreamInfo) (Result, error) {
	_, _ = io.Copy(io.Discard, r) // a real converter consumes the stream
	if f.err != nil {
		return Result{}, f.err
	}
	return Result{Markdown: f.out}, nil
}

func TestNormalizeWhitespaceAndBlankLines(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"trailing spaces and tabs", "a   \nb\t\n", "a\nb"},
		{"crlf to lf", "a\r\nb\r\n", "a\nb"},
		{"collapse many blank lines", "a\n\n\n\n\nb", "a\n\nb"},
		{"strip leading and trailing blank lines, keep content indent", "\n\n  a  \n\n", "  a"},
		{"preserve a single blank line", "a\n\nb", "a\n\nb"},
		{"empty stays empty", "", ""},
	}
	for _, c := range cases {
		if got := normalize(c.in); got != c.want {
			t.Errorf("%s: normalize(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestNormalizeRewritesBulletGlyphsToDash(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"unicode bullet", "• Hello", "- Hello"},
		{"black square", "▪ Foo", "- Foo"},
		{"indent preserved for nesting", "  • Indented", "  - Indented"},
		{"private use area (resume bullet)", "\uF0B7 Pua bullet", "- Pua bullet"},
		{"no space after glyph stays as-is", "•Hello", "•Hello"},
		{"glyph not at start stays as-is", "Hello • world", "Hello • world"},
	}
	for _, c := range cases {
		if got := normalize(c.in); got != c.want {
			t.Errorf("%s: normalize(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestNormalizeFoldsExoticSpacesAndDropsInvisibles(t *testing.T) {
	in := "Hello\u00A0world\u200B test\u3000end"
	want := "Hello world test end"
	if got := normalize(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizePreservesContentInCodeFences(t *testing.T) {
	// Bullet glyphs and NBSP inside a fenced code block must be left intact.
	in := "before\n\n```python\n• keep me\nthen\u00A0space\n```\n\n• after"
	got := normalize(in)
	if !strings.Contains(got, "• keep me") || !strings.Contains(got, "then\u00A0space") {
		t.Fatalf("fence content was modified:\n%s", got)
	}
	if !strings.Contains(got, "- after") {
		t.Fatalf("post-fence bullet not normalized:\n%s", got)
	}
}

func TestNormalizeWrapsImplausiblyLongLineAtSentenceBoundaries(t *testing.T) {
	sentence := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. "
	in := strings.Repeat(sentence, 20) // ~2300 chars, way over threshold
	got := normalize(in)
	lines := strings.Split(got, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected the long line to be split into multiple lines, got %d:\n%s", len(lines), got)
	}
	for _, ln := range lines {
		if len(ln) > longLineThreshold+200 {
			t.Fatalf("line still too long (%d chars): %q", len(ln), ln)
		}
	}
}

func TestNormalizeShortLinesUntouched(t *testing.T) {
	// Lines under the threshold must not be touched even if they end with ". A...".
	in := "Hello world. And then we did things. The end."
	if got := normalize(in); got != in {
		t.Fatalf("short line was modified:\n got: %q\nwant: %q", got, in)
	}
}

func TestConvertPrefersLowerPriority(t *testing.T) {
	reg := &Registry{}
	reg.Register(fakeConv{accept: true, out: "generic"}, 10)
	reg.Register(fakeConv{accept: true, out: "specific"}, 0)
	res, err := reg.Convert(strings.NewReader("x"), []StreamInfo{{}})
	if err != nil || res.Markdown != "specific" {
		t.Fatalf("got (%q, %v), want specific", res.Markdown, err)
	}
}

func TestConvertFallsThroughWhenConverterErrors(t *testing.T) {
	reg := &Registry{}
	// Higher-priority converter accepts but fails; the next accepting one wins.
	reg.Register(fakeConv{accept: true, out: "second"}, 5)
	reg.Register(fakeConv{accept: true, err: errors.New("boom")}, 0)
	res, err := reg.Convert(strings.NewReader("x"), []StreamInfo{{}})
	if err != nil || res.Markdown != "second" {
		t.Fatalf("got (%q, %v), want fall-through to second", res.Markdown, err)
	}
}

func TestConvertUnsupportedWhenNoneAccept(t *testing.T) {
	reg := &Registry{}
	reg.Register(fakeConv{accept: false}, 0)
	if _, err := reg.Convert(strings.NewReader("x"), nil); err != ErrUnsupported {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestConvertSurfacesConverterErrorWhenAllAcceptingFail(t *testing.T) {
	reg := &Registry{}
	reg.Register(fakeConv{accept: true, err: errors.New("boom")}, 0)
	_, err := reg.Convert(strings.NewReader("x"), nil)
	if err == nil || err == ErrUnsupported {
		t.Fatalf("want the underlying converter error, got %v", err)
	}
}

func TestConvertPropagatesReadError(t *testing.T) {
	reg := &Registry{}
	reg.Register(fakeConv{accept: true, out: "x"}, 0)
	if _, err := reg.Convert(iotest.ErrReader(errors.New("read fail")), nil); err == nil {
		t.Fatal("expected a read error to propagate")
	}
}
