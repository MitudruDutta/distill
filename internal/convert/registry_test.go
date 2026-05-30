package convert

import (
	"io"
	"strings"
	"testing"
)

type fakeConv struct {
	accept bool
	out    string
}

func (f fakeConv) Accepts(StreamInfo) bool { return f.accept }
func (f fakeConv) Convert(r io.Reader, _ StreamInfo) (Result, error) {
	_, _ = io.Copy(io.Discard, r)
	return Result{Markdown: f.out}, nil
}

func TestDispatchPrefersLowerPriority(t *testing.T) {
	reg := &Registry{}
	reg.Register(fakeConv{accept: true, out: "generic"}, 10)
	reg.Register(fakeConv{accept: true, out: "specific"}, 0)

	res, err := reg.Convert(strings.NewReader("x"), []StreamInfo{{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Markdown != "specific" {
		t.Fatalf("want %q, got %q", "specific", res.Markdown)
	}
}

func TestUnsupportedWhenNoneAccept(t *testing.T) {
	reg := &Registry{}
	reg.Register(fakeConv{accept: false}, 0)
	if _, err := reg.Convert(strings.NewReader("x"), nil); err != ErrUnsupported {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestNormalize(t *testing.T) {
	got := normalize("a   \n\n\n\nb \n")
	if got != "a\n\nb" {
		t.Fatalf("normalize got %q", got)
	}
}
