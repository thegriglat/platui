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
	CAFile  string // path (already expanded); empty — system CAs only
}

// New returns a client trusting system CAs plus the certificates from CAFile.
func New(o Options) (*http.Client, error) {
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
