package httpclient

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tlsServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	p := filepath.Join(t.TempDir(), "ca.pem")
	b := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return srv, p
}

func TestContractHTTPClientCAFile(t *testing.T) {
	srv, p := tlsServer(t)
	c, err := New(Options{Timeout: 5 * time.Second, CAFile: p})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode)
	}
}

func TestContractHTTPClientNoCAFile(t *testing.T) {
	srv, _ := tlsServer(t)
	c, err := New(Options{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Get(srv.URL); err == nil || !strings.Contains(err.Error(), "x509") {
		t.Fatalf("want x509 error, got %v", err)
	}
}

func TestContractHTTPClientBadFile(t *testing.T) {
	dir := t.TempDir()
	empty, garbage := filepath.Join(dir, "e.pem"), filepath.Join(dir, "g.pem")
	os.WriteFile(empty, nil, 0o600)
	os.WriteFile(garbage, []byte("not a cert"), 0o600)
	for _, p := range []string{empty, garbage} {
		_, err := New(Options{CAFile: p})
		if err == nil || !strings.Contains(err.Error(), "http.ca_file: "+p+": no PEM certificates found") {
			t.Errorf("%s: %v", p, err)
		}
	}
	missing := filepath.Join(dir, "none.pem")
	_, err := New(Options{CAFile: missing})
	if err == nil || !strings.HasPrefix(err.Error(), "http.ca_file: "+missing+": ") {
		t.Fatal(err)
	}
}
