package convert

import "testing"

func TestGuessMapsExtensionToMime(t *testing.T) {
	g := Guess(StreamInfo{Extension: ".JSON"}, nil)[0]
	if g.Extension != ".json" {
		t.Errorf("extension not lowercased: %q", g.Extension)
	}
	if g.Mimetype != "application/json" {
		t.Errorf("ext->mime: got %q, want application/json", g.Mimetype)
	}
}

func TestGuessSniffsContentWhenNoHints(t *testing.T) {
	g := Guess(StreamInfo{}, []byte("<!DOCTYPE html><html><body>hi</body></html>"))[0]
	if g.Mimetype != "text/html" {
		t.Errorf("sniff: got %q, want text/html", g.Mimetype)
	}
}

func TestGuessKeepsExplicitMimeOverExtension(t *testing.T) {
	g := Guess(StreamInfo{Mimetype: "application/json", Extension: ".csv"}, nil)[0]
	if g.Mimetype != "application/json" {
		t.Errorf("explicit mime should be kept, got %q", g.Mimetype)
	}
}

func TestExtensionOfLowercasesAndHandlesNone(t *testing.T) {
	if got := ExtensionOf("/path/IMG.PNG"); got != ".png" {
		t.Errorf("ExtensionOf = %q, want .png", got)
	}
	if got := ExtensionOf("noext"); got != "" {
		t.Errorf("ExtensionOf(noext) = %q, want empty", got)
	}
}
