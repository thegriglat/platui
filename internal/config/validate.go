package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{[^{}]*\}`)

func isHTTP(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// checkTemplate verifies that every placeholder of tpl is in allowed and, if
// needOne, that at least one is present.
func checkTemplate(key, tpl string, allowed []string, needOne bool) error {
	n := 0
	for _, p := range placeholderRe.FindAllString(tpl, -1) {
		if !contains(allowed, p) {
			return fmt.Errorf("%s: unknown placeholder %s", key, p)
		}
		n++
	}
	if needOne && n == 0 {
		return fmt.Errorf("%s must contain at least one of %s", key, strings.Join(allowed, ", "))
	}
	return nil
}

// Validate checks config invariants. It never includes the API key in errors.
func (c *Config) Validate() error {
	return c.validate(true)
}

func (c *Config) validate(needKey bool) error {
	if c.APIKey != "" && c.APIKeyEnv != "" {
		return errors.New("api_key and api_key_env are both set; use only one")
	}
	if needKey && c.APIKey == "" && c.APIKeyEnv == "" {
		return fmt.Errorf("neither api_key nor api_key_env is set: %w", ErrNoAPIKey)
	}
	if needKey && c.APIKeyEnv != "" && getenv(c.APIKeyEnv) == "" {
		return fmt.Errorf("environment variable %s (api_key_env) is empty or not set: %w", c.APIKeyEnv, ErrNoAPIKey)
	}
	if strings.TrimSpace(c.Workspace) == "" {
		return errors.New("workspace is required")
	}
	if c.BaseURL != "" && !isHTTP(c.BaseURL) {
		return fmt.Errorf("base_url must start with http:// or https://: %q", c.BaseURL)
	}
	if c.WebURL != "" && !isHTTP(c.WebURL) {
		return fmt.Errorf("web_url must start with http:// or https://: %q", c.WebURL)
	}
	if err := c.validateHTTP(); err != nil {
		return err
	}
	return c.validateV2()
}
