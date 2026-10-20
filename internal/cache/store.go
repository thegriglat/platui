// Package cache is an on-disk cache (contract C8 v1).
package cache

import (
	"fmt"
	"strings"
	"time"
)

// Store holds JSON-serializable values by key.
type Store interface {
	// Load: missing file, corrupt file or foreign version → ok=false, err=nil.
	Load(key string, v any) (saved time.Time, ok bool, err error)
	Save(key string, v any) error
}

const version = 1

// checkKey validates a key: segments separated by "/", characters [a-zA-Z0-9._-]+, no "." or "..".
func checkKey(key string) error {
	if key == "" {
		return fmt.Errorf("cache: empty key")
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("cache: invalid key %q", key)
		}
		for _, r := range seg {
			ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
			if !ok {
				return fmt.Errorf("cache: invalid key %q", key)
			}
		}
	}
	return nil
}
