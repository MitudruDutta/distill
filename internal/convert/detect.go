package convert

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Guess builds an ordered list of StreamInfo guesses from a base guess (derived
// from filename/flags) and a peek at the leading bytes of the content. The base
// guess takes precedence over content sniffing.
func Guess(base StreamInfo, peek []byte) []StreamInfo {
	g := base
	g.Extension = strings.ToLower(g.Extension)

	if g.Mimetype == "" && g.Extension != "" {
		if m := mime.TypeByExtension(g.Extension); m != "" {
			g.Mimetype = stripParams(m)
		}
	}
	if g.Mimetype == "" && len(peek) > 0 {
		g.Mimetype = stripParams(http.DetectContentType(peek))
	}
	return []StreamInfo{g}
}

// ExtensionOf returns the lowercased extension (including the dot) of a path.
func ExtensionOf(name string) string {
	return strings.ToLower(filepath.Ext(name))
}

func stripParams(m string) string {
	if i := strings.IndexByte(m, ';'); i >= 0 {
		return strings.TrimSpace(m[:i])
	}
	return m
}
