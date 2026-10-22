package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Defaults and limits for the http section (C1 v7).
const (
	DefaultHTTPTimeout = 30 * time.Second
	MinHTTPTimeout     = time.Second
	MaxHTTPTimeout     = 5 * time.Minute
	DefaultHTTPRetries = 3
	MaxHTTPRetries     = 10
)

// HTTP configures Plane API requests (C1 v7). nil fields mean defaults.
type HTTP struct {
	Timeout *time.Duration    `yaml:"timeout"`
	Retries *int              `yaml:"retries"`
	CAFile  string            `yaml:"ca_file"`
	Headers map[string]string `yaml:"headers"`
}

func (c *Config) validateHTTP() error {
	if t := c.HTTP.Timeout; t != nil && (*t < MinHTTPTimeout || *t > MaxHTTPTimeout) {
		return fmt.Errorf("http.timeout must be between %s and %s: %s", MinHTTPTimeout, MaxHTTPTimeout, *t)
	}
	if n := c.HTTP.Retries; n != nil && (*n < 0 || *n > MaxHTTPRetries) {
		return fmt.Errorf("http.retries must be between 0 and %d: %d", MaxHTTPRetries, *n)
	}
	for k := range c.HTTP.Headers {
		if strings.TrimSpace(k) == "" || strings.ContainsAny(k, " \t\r\n:") {
			return fmt.Errorf("http.headers: invalid header name %q", k)
		}
	}
	return nil
}

// HTTPHeaders returns extra headers for every Plane request (e.g. Cloudflare Access service token).
func (c *Config) HTTPHeaders() map[string]string { return c.HTTP.Headers }

// HTTPTimeout returns the per-request timeout.
func (c *Config) HTTPTimeout() time.Duration {
	if c.HTTP.Timeout == nil {
		return DefaultHTTPTimeout
	}
	return *c.HTTP.Timeout
}

// HTTPRetries returns the number of retries for 429/503.
func (c *Config) HTTPRetries() int {
	if c.HTTP.Retries == nil {
		return DefaultHTTPRetries
	}
	return *c.HTTP.Retries
}

// HTTPCAFile returns http.ca_file with a leading "~/" expanded; empty if unset (C1 v8).
func (c *Config) HTTPCAFile() string {
	p := c.HTTP.CAFile
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}
