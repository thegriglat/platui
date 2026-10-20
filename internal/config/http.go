package config

import (
	"fmt"
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
	Timeout *time.Duration `yaml:"timeout"`
	Retries *int           `yaml:"retries"`
}

func (c *Config) validateHTTP() error {
	if t := c.HTTP.Timeout; t != nil && (*t < MinHTTPTimeout || *t > MaxHTTPTimeout) {
		return fmt.Errorf("http.timeout must be between %s and %s: %s", MinHTTPTimeout, MaxHTTPTimeout, *t)
	}
	if n := c.HTTP.Retries; n != nil && (*n < 0 || *n > MaxHTTPRetries) {
		return fmt.Errorf("http.retries must be between 0 and %d: %d", MaxHTTPRetries, *n)
	}
	return nil
}

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
