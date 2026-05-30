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
		{"strip leading and trailing blanks", "\n\n  a  \n\n", "a"},
		{"preserve a single blank line", "a\n\nb", "a\n\nb"},
		{"empty stays empty", "", ""},
	}
	for _, c := range cases {
		if got := normalize(c.in); got != c.want {
			t.Errorf("%s: normalize(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
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
