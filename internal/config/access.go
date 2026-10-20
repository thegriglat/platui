package config

import (
	"os"
	"path/filepath"
	"time"
)

func getenv(k string) string { return os.Getenv(k) }

// Key returns the resolved API key.
func (c *Config) Key() string {
	if c.APIKey == "" && c.APIKeyEnv != "" {
		return getenv(c.APIKeyEnv)
	}
	return c.APIKey
}

// CheckoutEnabled reports whether branch creation should switch to the branch.
func (c *Config) CheckoutEnabled() bool {
	return c.Branch.Checkout == nil || *c.Branch.Checkout
}

// ColorsDisabled reports whether output must be uncolored (theme.no_color or NO_COLOR).
func (c *Config) ColorsDisabled() bool {
	return c.Theme.NoColor || getenv("NO_COLOR") != ""
}

// EditorCmd returns editor, else $VISUAL, else $EDITOR, else "vi".
func (c *Config) EditorCmd() string {
	for _, e := range []string{c.Editor, getenv("VISUAL"), getenv("EDITOR")} {
		if e != "" {
			return e
		}
	}
	return "vi"
}

// CacheDir returns cache.dir, else $XDG_CACHE_HOME/platui, else ~/.cache/platui.
func (c *Config) CacheDir() (string, error) {
	if c.Cache.Dir != "" {
		return c.Cache.Dir, nil
	}
	if x := getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "platui"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errHome(err)
	}
	return filepath.Join(home, ".cache", "platui"), nil
}

// CacheEnabled reports whether the disk cache is on (default true).
func (c *Config) CacheEnabled() bool { return c.Cache.Enabled == nil || *c.Cache.Enabled }

// RefreshInterval returns the polling interval; 0 means off.
func (c *Config) RefreshInterval() time.Duration {
	if c.Refresh.Interval == nil {
		return DefaultRefresh
	}
	return *c.Refresh.Interval
}

// NotifyEnabled reports whether change notifications are on (default true).
func (c *Config) NotifyEnabled() bool { return c.Refresh.Notify == nil || *c.Refresh.Notify }
