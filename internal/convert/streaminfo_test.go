package convert

import "testing"

func TestStreamInfoMergeOverlaysNonEmptyFields(t *testing.T) {
	base := StreamInfo{Mimetype: "text/plain", Extension: ".txt", Filename: "a.txt"}
	got := base.Merge(StreamInfo{Mimetype: "text/html", Charset: "utf-8"})

	if got.Mimetype != "text/html" {
		t.Errorf("mimetype should be overridden: got %q", got.Mimetype)
	}
	if got.Charset != "utf-8" {
		t.Errorf("charset should be set: got %q", got.Charset)
	}
	if got.Extension != ".txt" || got.Filename != "a.txt" {
		t.Errorf("non-empty base fields must be preserved: %+v", got)
	}
}
