package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	converters "github.com/MitudruDutta/distill/internal/converters/src"
)

func TestServeConvertsCSV(t *testing.T) {
	srv := httptest.NewServer(Handler(converters.Default(), "", 1<<20))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/convert?ext=csv", "text/csv", strings.NewReader("a,b\n1,2\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "| a | b |") {
		t.Fatalf("body:\n%s", body)
	}
}

func TestServeHealthz(t *testing.T) {
	srv := httptest.NewServer(Handler(converters.Default(), "", 1<<20))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status %d", resp.StatusCode)
	}
}

func TestServeEnforcesToken(t *testing.T) {
	srv := httptest.NewServer(Handler(converters.Default(), "secret", 1<<20))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/convert?ext=csv", "text/csv", strings.NewReader("a\n1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("without token want 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/convert?ext=csv", strings.NewReader("a\n1\n"))
	req.Header.Set("Authorization", "Bearer secret")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("with token want 200, got %d", resp2.StatusCode)
	}
}

func TestNewServerRefusesNonLoopbackWithoutToken(t *testing.T) {
	if _, err := NewServer(converters.Default(), ServeOptions{Addr: "0.0.0.0:0"}); err == nil {
		t.Fatal("expected refusal to bind a non-loopback address without a token")
	}
	if _, err := NewServer(converters.Default(), ServeOptions{Addr: "127.0.0.1:0"}); err != nil {
		t.Fatalf("loopback bind should be allowed: %v", err)
	}
}
