package app

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/MitudruDutta/distill/internal/convert"
)

// ServeOptions configures the HTTP conversion server.
type ServeOptions struct {
	Addr     string // host:port
	Token    string // bearer token; REQUIRED when Addr is non-loopback
	MaxBytes int64  // max request body (0 => 32 MiB)
}

// NewServer builds an HTTP server exposing POST /convert and GET /healthz. It
// refuses to bind a non-loopback address without an auth token, so it is never
// silently exposed to the network.
func NewServer(reg *convert.Registry, opts ServeOptions) (*http.Server, error) {
	if !isLoopback(opts.Addr) && opts.Token == "" {
		return nil, errors.New("serve: refusing to bind a non-loopback address without an auth token (set --token or DISTILL_TOKEN)")
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 32 << 20
	}
	return &http.Server{
		Addr:              opts.Addr,
		Handler:           Handler(reg, opts.Token, opts.MaxBytes),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}, nil
}

// Handler returns the conversion HTTP handler. When token is non-empty,
// /convert requires "Authorization: Bearer <token>" or "X-Auth-Token: <token>".
func Handler(reg *convert.Registry, token string, maxBytes int64) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok")
	})
	mux.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if token != "" && !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "request too large or unreadable", http.StatusRequestEntityTooLarge)
			return
		}
		base := convert.StreamInfo{Mimetype: r.Header.Get("Content-Type")}
		if ext := r.URL.Query().Get("ext"); ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			base.Extension = strings.ToLower(ext)
		}
		peek := data
		if len(peek) > 512 {
			peek = peek[:512]
		}
		res, err := reg.Convert(bytes.NewReader(data), convert.Guess(base, peek))
		if err != nil {
			http.Error(w, "could not convert: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if wantsJSON(r) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(res)
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		io.WriteString(w, res.Markdown)
	})
	return mux
}

func authorized(r *http.Request, token string) bool {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		if subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(h, "Bearer ")), []byte(token)) == 1 {
			return true
		}
	}
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Auth-Token")), []byte(token)) == 1
}

func wantsJSON(r *http.Request) bool {
	return r.URL.Query().Get("format") == "json" ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
}

// isLoopback reports whether addr binds only the loopback interface. An empty
// host (e.g. ":8080") binds all interfaces and is treated as non-loopback.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	switch host {
	case "localhost":
		return true
	case "":
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
