package app

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/MitudruDutta/distill/internal/convert"
)

// FetchOptions configures URL fetching. The zero value is safe defaults: a
// 30-second total timeout, a 32 MiB body cap, 5 redirect hops, the four
// standard schemes (http, https, file, data), and SSRF guards that reject
// loopback, private (RFC 1918 / RFC 4193), link-local (incl. cloud metadata
// 169.254.169.254), multicast, and unspecified IPs.
type FetchOptions struct {
	UserAgent    string
	Timeout      time.Duration
	MaxBytes     int64
	MaxRedirects int
	AllowSchemes []string

	// AllowLoopback disables the loopback-IP block. **Tests only.** Production
	// callers should leave this false; otherwise a hostile DNS rebind can
	// trick the fetcher into hitting localhost services.
	AllowLoopback bool
}

func (o *FetchOptions) setDefaults() {
	if o.UserAgent == "" {
		o.UserAgent = "distill/0.1"
	}
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.MaxBytes == 0 {
		o.MaxBytes = 32 << 20
	}
	if o.MaxRedirects == 0 {
		o.MaxRedirects = 5
	}
	if len(o.AllowSchemes) == 0 {
		o.AllowSchemes = []string{"http", "https", "file", "data"}
	}
}

// FetchURI fetches a URI (http://, https://, file://, data:) and returns the
// raw bytes plus a populated StreamInfo (mimetype, charset, filename,
// extension, original URL).
//
// HTTP/HTTPS connections are SSRF-guarded via a Dialer.Control callback that
// runs after DNS resolution and before connect, so each redirect/retry is
// re-validated and DNS rebinding cannot bypass the check.
func FetchURI(uri string, opts FetchOptions) ([]byte, convert.StreamInfo, error) {
	opts.setDefaults()
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, convert.StreamInfo{}, fmt.Errorf("invalid URI: %w", err)
	}
	if !schemeAllowed(parsed.Scheme, opts.AllowSchemes) {
		return nil, convert.StreamInfo{}, fmt.Errorf("scheme %q is not allowed", parsed.Scheme)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return fetchHTTP(parsed, opts)
	case "file":
		return fetchFile(parsed)
	case "data":
		return fetchData(uri)
	}
	return nil, convert.StreamInfo{}, fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
}

// IsURI reports whether s looks like one of the URI schemes FetchURI accepts.
func IsURI(s string) bool {
	for _, p := range []string{"http://", "https://", "file://", "data:"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func schemeAllowed(scheme string, allow []string) bool {
	s := strings.ToLower(scheme)
	for _, a := range allow {
		if strings.ToLower(a) == s {
			return true
		}
	}
	return false
}

// fetchHTTP performs an SSRF-guarded HTTP GET.
func fetchHTTP(u *url.URL, opts FetchOptions) ([]byte, convert.StreamInfo, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: makeSSRFControl(opts),
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= opts.MaxRedirects {
				return fmt.Errorf("too many redirects (>%d)", opts.MaxRedirects)
			}
			if !schemeAllowed(req.URL.Scheme, opts.AllowSchemes) {
				return fmt.Errorf("redirect to disallowed scheme %q", req.URL.Scheme)
			}
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, convert.StreamInfo{}, err
	}
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, convert.StreamInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, convert.StreamInfo{}, fmt.Errorf("http %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body := http.MaxBytesReader(nil, resp.Body, opts.MaxBytes)
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, convert.StreamInfo{}, fmt.Errorf("body: %w", err)
	}

	info := convert.StreamInfo{URL: resp.Request.URL.String()}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		if mt, params, err := mime.ParseMediaType(ct); err == nil {
			info.Mimetype = mt
			if cs, ok := params["charset"]; ok {
				info.Charset = cs
			}
		} else {
			info.Mimetype = ct
		}
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fname, ok := params["filename"]; ok {
				info.Filename = fname
				info.Extension = strings.ToLower(filepath.Ext(fname))
			}
		}
	}
	if info.Filename == "" {
		if base := filepath.Base(resp.Request.URL.Path); base != "" && base != "/" && base != "." {
			info.Filename = base
			if info.Extension == "" {
				info.Extension = strings.ToLower(filepath.Ext(base))
			}
		}
	}
	return data, info, nil
}

// makeSSRFControl returns a Dialer.Control callback that rejects connections
// to disallowed IP ranges. Runs per-connection (so redirects and retries are
// re-validated) and after DNS resolution (so rebinding can't bypass).
func makeSSRFControl(opts FetchOptions) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("ssrf: bad address %q: %w", address, err)
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("ssrf: cannot parse %q as IP", host)
		}
		if isForbiddenIP(ip, opts.AllowLoopback) {
			return fmt.Errorf("ssrf: refusing to connect to %s", ip)
		}
		return nil
	}
}

// isForbiddenIP reports whether the IP lies in a range we refuse to dial.
// Categories: loopback (unless allowLoopback), private (RFC 1918 / RFC 4193),
// link-local (incl. cloud metadata 169.254.169.254), multicast, unspecified.
func isForbiddenIP(ip net.IP, allowLoopback bool) bool {
	if !allowLoopback && ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	return false
}

// fetchFile reads a file:// URI from the local filesystem. Host must be empty
// or "localhost".
func fetchFile(u *url.URL) ([]byte, convert.StreamInfo, error) {
	if u.Host != "" && u.Host != "localhost" {
		return nil, convert.StreamInfo{}, fmt.Errorf("file URI host must be empty or localhost (got %q)", u.Host)
	}
	p := u.Path
	if p == "" {
		p = u.Opaque
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, convert.StreamInfo{}, err
	}
	info := convert.StreamInfo{
		Filename:  filepath.Base(p),
		LocalPath: p,
		Extension: strings.ToLower(filepath.Ext(p)),
		URL:       u.String(),
	}
	return data, info, nil
}

// fetchData parses an RFC 2397 data: URI.
//
//	data:[<mediatype>][;base64],<data>
func fetchData(uri string) ([]byte, convert.StreamInfo, error) {
	rest := strings.TrimPrefix(uri, "data:")
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return nil, convert.StreamInfo{}, errors.New("malformed data URI: missing comma")
	}
	meta, payload := rest[:comma], rest[comma+1:]
	isBase64 := false
	if strings.HasSuffix(meta, ";base64") {
		isBase64 = true
		meta = strings.TrimSuffix(meta, ";base64")
	}

	var data []byte
	if isBase64 {
		dec, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, convert.StreamInfo{}, fmt.Errorf("data URI base64: %w", err)
		}
		data = dec
	} else {
		decoded, err := url.QueryUnescape(payload)
		if err != nil {
			return nil, convert.StreamInfo{}, fmt.Errorf("data URI payload: %w", err)
		}
		data = []byte(decoded)
	}

	info := convert.StreamInfo{URL: uri}
	if meta == "" {
		info.Mimetype = "text/plain"
		info.Charset = "us-ascii"
	} else if mt, params, err := mime.ParseMediaType(meta); err == nil {
		info.Mimetype = mt
		info.Charset = params["charset"]
	} else {
		info.Mimetype = meta
	}
	return data, info, nil
}
