package converters

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/MitudruDutta/distill/internal/convert"
)

// EML converts an RFC 822 e-mail: key headers become a small block, then the
// body. For multipart messages, text/plain is preferred, falling back to
// text/html (converted to Markdown). Base64 / quoted-printable parts are decoded.
type EML struct{}

func (EML) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".eml" || info.Mimetype == "message/rfc822"
}

func (EML) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return convert.Result{}, err
	}

	var b strings.Builder
	for _, h := range []string{"From", "To", "Subject", "Date"} {
		if v := msg.Header.Get(h); v != "" {
			fmt.Fprintf(&b, "**%s:** %s\n", h, decodeHeaderWord(v))
		}
	}
	b.WriteString("\n")

	body, err := emlBody(msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), msg.Body)
	if err != nil {
		return convert.Result{}, err
	}
	b.WriteString(body)
	return convert.Result{Markdown: strings.TrimSpace(b.String()), Title: decodeHeaderWord(msg.Header.Get("Subject"))}, nil
}

func emlBody(ctype, cte string, r io.Reader) (string, error) {
	mediaType, params, _ := mime.ParseMediaType(ctype)
	if strings.HasPrefix(mediaType, "multipart/") && params["boundary"] != "" {
		return multipartBody(r, params["boundary"]), nil
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return renderPart(mediaType, cte, data), nil
}

func multipartBody(r io.Reader, boundary string) string {
	mr := multipart.NewReader(r, boundary)
	var plain, html string
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		mediaType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		data, _ := io.ReadAll(p)
		rendered := renderPart(mediaType, p.Header.Get("Content-Transfer-Encoding"), data)
		switch {
		case strings.HasPrefix(mediaType, "text/plain") && plain == "":
			plain = rendered
		case strings.HasPrefix(mediaType, "text/html") && html == "":
			html = rendered
		}
	}
	if plain != "" {
		return plain
	}
	return html
}

func renderPart(mediaType, cte string, data []byte) string {
	data = decodeCTE(cte, data)
	if strings.HasPrefix(mediaType, "text/html") {
		if md, err := htmltomarkdown.ConvertString(string(data)); err == nil {
			return strings.TrimSpace(md)
		}
	}
	return strings.TrimSpace(string(data))
}

func decodeCTE(enc string, data []byte) []byte {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "base64":
		if dec, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data))); err == nil {
			return dec
		}
	case "quoted-printable":
		if dec, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(data))); err == nil {
			return dec
		}
	}
	return data
}

func decodeHeaderWord(s string) string {
	if out, err := (&mime.WordDecoder{}).DecodeHeader(s); err == nil {
		return out
	}
	return s
}
