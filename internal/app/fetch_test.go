package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchHTTPHappyPathSetsMimeAndExtension(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte("a,b\n1,2\n"))
	}))
	defer srv.Close()

	data, info, err := FetchURI(srv.URL+"/sales.csv", FetchOptions{AllowLoopback: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a,b\n1,2\n" {
		t.Fatalf("body: %q", data)
	}
	if info.Mimetype != "text/csv" {
		t.Errorf("mimetype = %q, want text/csv", info.Mimetype)
	}
	if info.Charset != "utf-8" {
		t.Errorf("charset = %q, want utf-8", info.Charset)
	}
	if info.Extension != ".csv" {
		t.Errorf("extension = %q, want .csv", info.Extension)
	}
	if info.Filename != "sales.csv" {
		t.Errorf("filename = %q, want sales.csv", info.Filename)
	}
}

func TestFetchHTTPHonorsContentDispositionFilename(t *testing.T) {

func TestFetchHTTPSendsCustomUserAgent(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	if _, _, err := FetchURI(srv.URL, FetchOptions{AllowLoopback: true, UserAgent: "test-agent/9.9"}); err != nil {
		t.Fatal(err)
	}
	if seen != "test-agent/9.9" {
		t.Fatalf("server saw User-Agent %q, want test-agent/9.9", seen)
	}
}

func TestFetchHTTPDefaultUserAgentWhenUnset(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	if _, _, err := FetchURI(srv.URL, FetchOptions{AllowLoopback: true}); err != nil {
		t.Fatal(err)
	}
	if seen == "" {
		t.Fatal("default User-Agent must be set")
	}
}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		_, _ = w.Write([]byte("%PDF-1.4 stub"))
	}))
	defer srv.Close()

	_, info, err := FetchURI(srv.URL+"/dl", FetchOptions{AllowLoopback: true})
	if err != nil {
		t.Fatal(err)
	}
	if info.Filename != "report.pdf" {
		t.Errorf("filename = %q, want report.pdf", info.Filename)
	}
	if info.Extension != ".pdf" {
		t.Errorf("extension = %q, want .pdf", info.Extension)
	}
}

func TestFetchHTTPRefusesLoopbackByDefault(t *testing.T) {
	// AllowLoopback is false by default → 127.0.0.1 must be rejected.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should not reach"))
	}))
	defer srv.Close()

	_, _, err := FetchURI(srv.URL, FetchOptions{})
	if err == nil {
		t.Fatal("expected SSRF refusal of loopback")
	}
	if !strings.Contains(err.Error(), "ssrf") {
		t.Errorf("error should mention ssrf: %v", err)
	}
}

func TestFetchHTTPRefusesLinkLocalCloudMetadataIP(t *testing.T) {
	// 169.254.169.254 is the AWS / GCE / Azure metadata IP. Must be refused
	// before any connection attempt is made.
	_, _, err := FetchURI("http://169.254.169.254/latest/meta-data/", FetchOptions{})
	if err == nil {
		t.Fatal("expected SSRF refusal of cloud-metadata IP")
	}
	if !strings.Contains(err.Error(), "ssrf") && !strings.Contains(err.Error(), "169.254") {
		t.Errorf("error should mention ssrf or the IP: %v", err)
	}
}

func TestFetchHTTPRefusesPrivateRFC1918IP(t *testing.T) {
	for _, addr := range []string{"http://10.0.0.1/", "http://192.168.1.1/", "http://172.16.0.1/"} {
		_, _, err := FetchURI(addr, FetchOptions{})
		if err == nil {
			t.Errorf("%s: expected SSRF refusal", addr)
		}
	}
}

func TestFetchSchemeAllowlist(t *testing.T) {
	for _, bad := range []string{"ftp://example.com/", "gopher://example.com/", "javascript:alert(1)"} {
		if _, _, err := FetchURI(bad, FetchOptions{}); err == nil {
			t.Errorf("%s: expected scheme rejection", bad)
		}
	}
}

func TestFetchHTTPBodyCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 10000))
	}))
	defer srv.Close()
	_, _, err := FetchURI(srv.URL, FetchOptions{AllowLoopback: true, MaxBytes: 100})
	if err == nil {
		t.Fatal("expected body cap to fire")
	}
}

func TestFetchHTTPRedirectCap(t *testing.T) {
	hops := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops++
		http.Redirect(w, r, srv.URL+"/r", http.StatusFound)
	}))
	defer srv.Close()
	_, _, err := FetchURI(srv.URL, FetchOptions{AllowLoopback: true, MaxRedirects: 2})
	if err == nil {
		t.Fatal("expected redirect cap to fire")
	}
	if !strings.Contains(err.Error(), "redirect") {
		t.Errorf("error should mention redirects: %v", err)
	}
}

func TestFetchHTTPRedirectToDisallowedSchemeRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "ftp://example.com/")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	_, _, err := FetchURI(srv.URL, FetchOptions{AllowLoopback: true})
	if err == nil {
		t.Fatal("expected refusal of redirect to ftp")
	}
}

func TestFetchFileURI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.csv")
	if err := os.WriteFile(p, []byte("x,y\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, info, err := FetchURI("file://"+p, FetchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "x,y\n1,2\n" {
		t.Fatalf("data: %q", data)
	}
	if info.Extension != ".csv" {
		t.Errorf("extension = %q, want .csv", info.Extension)
	}
	if info.Filename != "sample.csv" {
		t.Errorf("filename = %q, want sample.csv", info.Filename)
	}
}

func TestFetchFileURIRejectsRemoteHost(t *testing.T) {
	if _, _, err := FetchURI("file://example.com/etc/passwd", FetchOptions{}); err == nil {
		t.Fatal("expected refusal of file URI with non-localhost host")
	}
}

func TestFetchDataURIBase64(t *testing.T) {
	// "a,b\n1,2\n" base64-encoded
	data, info, err := FetchURI("data:text/csv;base64,YSxiCjEsMgo=", FetchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a,b\n1,2\n" {
		t.Fatalf("data: %q", data)
	}
	if info.Mimetype != "text/csv" {
		t.Errorf("mimetype = %q", info.Mimetype)
	}
}

func TestFetchDataURIPlainURLEncoded(t *testing.T) {
	data, info, err := FetchURI("data:text/plain,hello%20world", FetchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Fatalf("data: %q", data)
	}
	if info.Mimetype != "text/plain" {
		t.Errorf("mimetype = %q", info.Mimetype)
	}
}

func TestFetchDataURIDefaultMime(t *testing.T) {
	_, info, err := FetchURI("data:,plain%20text", FetchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if info.Mimetype != "text/plain" {
		t.Errorf("default mimetype = %q, want text/plain", info.Mimetype)
	}
}

func TestIsURI(t *testing.T) {
	for _, s := range []string{"http://x", "https://x", "file:///x", "data:text/plain,x"} {
		if !IsURI(s) {
			t.Errorf("IsURI(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"/path/to/file", "relative/path", "ftp://x", ""} {
		if IsURI(s) {
			t.Errorf("IsURI(%q) = true, want false", s)
		}
	}
}
