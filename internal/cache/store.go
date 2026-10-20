// Package cache — кэш на диске (контракт C8 v1).
package cache

import (
	"fmt"
	"strings"
	"time"
)

// Store — хранилище JSON-сериализуемых значений по ключу.
type Store interface {
	// Load: нет файла, битый файл или чужая версия → ok=false, err=nil.
	Load(key string, v any) (saved time.Time, ok bool, err error)
	Save(key string, v any) error
}

const version = 1

// checkKey проверяет ключ: сегменты через «/», символы [a-zA-Z0-9._-]+, без "." и "..".
func checkKey(key string) error {
	if key == "" {
		return fmt.Errorf("cache: пустой ключ")
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("cache: недопустимый ключ %q", key)
		}
		for _, r := range seg {
			ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
			if !ok {
				return fmt.Errorf("cache: недопустимый ключ %q", key)
			}
		}
	}
	return nil
}
