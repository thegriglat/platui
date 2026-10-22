// Package httpclient builds the HTTP client for Plane (C13 v1).
package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Options configures New.
type Options struct {
	Timeout time.Duration
	CAFile  string            // path (already expanded); empty — system CAs only
	Headers map[string]string // added to every request (e.g. CF-Access-Client-Id/Secret)
}

// New returns a client trusting system CAs plus the certificates from CAFile
// and sending Headers with every request.
func New(o Options) (*http.Client, error) {
	c, err := newBase(o)
	if err != nil {
		return nil, err
	}
	if len(o.Headers) > 0 {
		base := c.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		c.Transport = headerTransport{base: base, h: o.Headers}
	}
	return c, nil
}

// headerTransport sets fixed headers on each request.
type headerTransport struct {
	base http.RoundTripper
	h    map[string]string
}

func (t headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range t.h {
		r.Header.Set(k, v)
	}
	return t.base.RoundTrip(r)
}

func newBase(o Options) (*http.Client, error) {
	c := &http.Client{Timeout: o.Timeout}
	if o.CAFile == "" {
		return c, nil
	}
	pem, err := os.ReadFile(o.CAFile)
	if err != nil {
		return nil, fmt.Errorf("http.ca_file: %s: %s", o.CAFile, reason(err))
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("http.ca_file: %s: no PEM certificates found", o.CAFile)
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	c.Transport = tr
	return c, nil
}

// reason strips the path from *PathError so it is not repeated.
func reason(err error) string {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err.Error()
	}
	return err.Error()
}
