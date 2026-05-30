package converters

import (
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestJSONPrettyPrintsPreservingKeyOrder(t *testing.T) {
	res, err := (JSON{}).Convert(strings.NewReader(`{"b":1,"a":[2,3]}`), convert.StreamInfo{Extension: ".json"})
	if err != nil {
		t.Fatal(err)
	}
	want := "```json\n{\n  \"b\": 1,\n  \"a\": [\n    2,\n    3\n  ]\n}\n```"
	if res.Markdown != want {
		t.Fatalf("\n got: %q\nwant: %q", res.Markdown, want)
	}
}

func TestJSONMalformedReturnsError(t *testing.T) {
	if _, err := (JSON{}).Convert(strings.NewReader("{not json"), convert.StreamInfo{Extension: ".json"}); err == nil {
		t.Fatal("expected an error on malformed JSON")
	}
}
