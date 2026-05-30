package converters_test

import (
	. "github.com/MitudruDutta/distill/internal/converters/src"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
)

func TestFencedWrapsContentWithLanguageTag(t *testing.T) {
	res, _ := (Fenced{Lang: "yaml"}).Convert(strings.NewReader("a: 1\n\n"), convert.StreamInfo{})
	if res.Markdown != "```yaml\na: 1\n```" {
		t.Fatalf("yaml fence: %q", res.Markdown)
	}
}

func TestFencedUsesLongerFenceWhenContentHasBackticks(t *testing.T) {
	res, _ := (Fenced{Lang: "md"}).Convert(strings.NewReader("```\ncode\n```"), convert.StreamInfo{})
	if !strings.HasPrefix(res.Markdown, "````md\n") || !strings.HasSuffix(res.Markdown, "\n````") {
		t.Fatalf("expected a 4-backtick fence, got:\n%s", res.Markdown)
	}
}
