package todo

import (
	"os"
	"path/filepath"

	"github.com/google/renameio/v2"
)

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return renameio.WriteFile(path, data, 0o644)
}
